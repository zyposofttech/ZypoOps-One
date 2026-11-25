package helper

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"maps"
	"time"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-policy"
	contracts "zyposoft.local/operations/packages/management-contracts"
	pl "zyposoft.local/operations/packages/product-local"
	"zyposoft.local/operations/packages/signing-go"
)

func OpenSimulation(path string, c policy.Config, uid, gid uint32, now func() time.Time, products pl.Registry) (*Helper, error) {
	products = maps.Clone(products)
	c.LocalCheck = products.Check
	h, err := Open(path, c, uid, gid, now)
	if err != nil {
		return nil, err
	}
	_, err = h.db.Exec(`CREATE TABLE IF NOT EXISTS deployment_execution(job TEXT PRIMARY KEY REFERENCES simulated_deployment(job), bundle TEXT NOT NULL, execution TEXT NOT NULL UNIQUE, result BLOB); CREATE TABLE IF NOT EXISTS refused_bundle(binding TEXT PRIMARY KEY)`)
	if err != nil {
		h.Close()
		return nil, err
	}
	h.products = products
	return h, nil
}
func (h *Helper) dispatch(ctx context.Context, wire []byte) deployment.Reply {
	deny := deployment.Reply{Status: deployment.Denied}
	retry := deployment.Reply{Status: deployment.Retry}
	raw, err := contracts.Canonical(wire)
	if err != nil {
		return deny
	}
	var req deployment.Request
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&req) != nil || (req.Operation != "lookup" && req.Operation != "execute" && req.Operation != "abort") || len(req.Bundle.Grants) > 8 {
		return deny
	}
	b := req.Bundle
	if _, err = signing.Verify("MC-03", b.Job, h.cfg.Scope, h.cfg.Trust); err != nil {
		return deny
	}
	var job contracts.MC03
	if json.Unmarshal(b.Job, &job) != nil {
		return deny
	}
	j := job.Body
	if j.CommandType != "DEPLOY_RELEASE" || j.AgentId != h.cfg.Agent || j.InstallationId != h.cfg.Installation {
		return deny
	}
	var p contracts.DeployReleaseParams
	if json.Unmarshal(j.Params, &p) != nil {
		return deny
	}
	binding, err := deployment.BundleDigest(b)
	if err != nil {
		return deny
	}
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return retry
	}
	defer tx.Rollback()
	now := h.now().UTC()
	blocked, err := observeTime(tx, now)
	if err != nil {
		return retry
	}
	if blocked {
		_ = tx.Commit() // Keep the rollback latch even when denying recovery.
		return retry
	}
	var refused int
	if err = tx.QueryRow("SELECT count(*) FROM refused_bundle WHERE binding=?", binding).Scan(&refused); err != nil {
		return retry
	}
	if refused != 0 {
		if tx.Commit() != nil {
			return retry
		}
		return deployment.Reply{Status: deployment.Refused}
	}
	refuse := func() deployment.Reply {
		if _, err := tx.Exec("INSERT INTO refused_bundle VALUES(?) ON CONFLICT DO NOTHING", binding); err != nil {
			return retry
		}
		if tx.Commit() != nil {
			return retry
		}
		return deployment.Reply{Status: deployment.Refused}
	}
	var original, execution string
	var saved []byte
	err = tx.QueryRow("SELECT bundle,execution,result FROM deployment_execution WHERE job=?", j.JobId).Scan(&original, &execution, &saved)
	if err != nil && err != sql.ErrNoRows {
		return retry
	}
	exists := err == nil
	if exists && original != binding {
		return deny
	}
	if len(saved) > 0 {
		var r deployment.Result
		if json.Unmarshal(saved, &r) != nil || r.Execution != execution || r.Validate(b) != nil || tx.Commit() != nil {
			return retry
		}
		return deployment.Reply{Status: deployment.Done, Result: &r}
	}
	if req.Operation == "lookup" && !exists {
		if tx.Commit() != nil {
			return retry
		}
		return deployment.Reply{Status: deployment.Absent}
	}
	if !exists {
		if req.Operation == "abort" {
			return refuse()
		}
		var pending int
		// A terminal receipt is immutable. A possibly unhealthy release still
		// holds the site until independently authorised recovery is available.
		if err = tx.QueryRow(`SELECT count(*) FROM deployment_execution WHERE result IS NULL OR
		    coalesce(json_extract(CAST(result AS TEXT),'$.state'),'') NOT IN ('SUCCEEDED','REJECTED')`).Scan(&pending); err != nil || pending != 0 {
			return retry
		}
		approved, err := h.cfg.AuthorizeContext(ctx, b, now)
		if err != nil {
			return refuse()
		}
		if _, err = tx.Exec("SAVEPOINT admission"); err != nil {
			return retry
		}
		if code := reserve(tx, approved); code != "SIMULATED" {
			if _, err = tx.Exec("ROLLBACK TO admission"); err != nil || code == "STORE_UNAVAILABLE" {
				return retry
			}
			return refuse()
		}
		execution = deployment.ExecutionID(binding)
		if _, err = tx.Exec("INSERT INTO deployment_execution(job,bundle,execution) VALUES(?,?,?)", j.JobId, binding, execution); err != nil {
			return retry
		}
	}
	if tx.Commit() != nil {
		return retry
	}
	return h.reconcile(ctx, req, job, p, execution)
}

// Only the exact reserved bundle reaches the product. A committed observation
// permits read-only reconciliation; ABSENT requires current authority to apply.
func (h *Helper) reconcile(ctx context.Context, req deployment.Request, job contracts.MC03, p contracts.DeployReleaseParams, execution string) deployment.Reply {
	deny := deployment.Reply{Status: deployment.Denied}
	retry := deployment.Reply{Status: deployment.Retry}
	b, j := req.Bundle, job.Body
	var manifest contracts.MC05
	if json.Unmarshal(b.Manifest, &manifest) != nil {
		return deny
	}
	app := h.products[p.ProductKey]
	if app == nil {
		return retry
	}
	activation := pl.Activation{Product: p.ProductKey, Job: j.JobId, Execution: execution, Artifact: p.ManifestSha256, Release: manifest.Body.ReleaseId, Expires: effectDeadline(b, j.ExpiresAt, manifest)}
	// A product commit may have survived interruption of the helper's reply or final write.
	observed, err := app.Lookup(ctx, activation)
	if err != nil {
		return retry
	}
	if observed.Phase == "ABSENT" && req.Operation == "lookup" {
		return deployment.Reply{Status: deployment.Pending}
	}

	if observed.Phase == "ABSENT" {
		if _, err = h.cfg.AuthorizeContext(ctx, b, h.now().UTC()); err != nil || req.Operation == "abort" {
			observed, err = app.Fence(ctx, activation)
			if err != nil {
				return retry
			}
		} else {
			observed, err = app.Apply(ctx, activation)
			if err != nil {
				return retry
			}
		}
	}
	digest, err := policy.Digest(b.Job)
	if err != nil {
		return deny
	}
	r := deployment.Result{Job: j.JobId, JobDigest: digest, Execution: execution, Product: p.ProductKey, Artifact: p.ManifestSha256}
	switch observed.Phase {
	case "BLOCKED":
		r.State, r.Stage, r.Code = "REJECTED", "BEFORE_EFFECT", "PRECONDITION_FAILED"
	case "APPLIED_ERROR":
		r.State, r.Stage, r.Code = "FAILED", "AFTER_EFFECT", "OTHER_CODED"
	case "APPLIED":
		ok, err := app.ReadyVersion(ctx, activation.Release)
		switch {
		case err != nil:
			r.State, r.Stage, r.Code = "UNKNOWN", "POST_VERIFY", "EXECUTION_UNKNOWN"
		case !ok:
			r.State, r.Stage, r.Code = "FAILED", "POST_VERIFY", "PRECONDITION_FAILED"
		default:
			r.State, r.Stage, r.Code = "SUCCEEDED", "VERIFIED", "NONE"
		}
	default:
		return retry
	}
	encoded, err := json.Marshal(r)
	if err != nil {
		return retry
	}
	changed, err := h.db.ExecContext(ctx, "UPDATE deployment_execution SET result=? WHERE job=? AND result IS NULL", encoded, j.JobId)
	if err != nil {
		return retry
	}
	if n, e := changed.RowsAffected(); e != nil || n != 1 {
		return retry
	}
	return deployment.Reply{Status: deployment.Done, Result: &r}
}

// Product-side authorization cannot outlive any proof checked by the helper.
func effectDeadline(b policy.Bundle, expires string, manifest contracts.MC05) string {
	floor, _ := time.Parse(time.RFC3339Nano, expires)
	take := func(v string, span time.Duration) {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			floor = time.Time{}
			return
		}
		if t = t.Add(span); t.Before(floor) {
			floor = t
		}
	}
	take(manifest.Body.ExpiresAt, 0)
	for _, raw := range b.Grants {
		var grant contracts.MC09
		if json.Unmarshal(raw, &grant) != nil {
			return ""
		}
		take(grant.Body.ValidUntil, 0)
	}
	if manifest.Body.BackupEvidenceRequired {
		var evidence contracts.MC14
		if json.Unmarshal(b.Backup, &evidence) != nil {
			return ""
		}
		take(evidence.Body.LastSuccessAt, 24*time.Hour)
	}
	return floor.UTC().Format(time.RFC3339Nano)
}
