package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-policy"
	contracts "zyposoft.local/operations/packages/management-contracts"
	pl "zyposoft.local/operations/packages/product-local"
	"zyposoft.local/operations/packages/signing-go"
)

func OpenConnected(path string, id Identity, c policy.Config, keys TestKeyProvider, log *slog.Logger, products pl.Registry, helper deployment.Helper) (*Agent, error) {
	if helper == nil || len(products) == 0 {
		return nil, errors.New("DEPLOYMENT_CONFIG")
	}
	c.LocalCheck = products.Check
	a, err := Open(path, id, c, keys, log)
	if err != nil {
		return nil, err
	}
	_, err = a.db.Exec(`CREATE TABLE IF NOT EXISTS pending_deployment(job_id TEXT PRIMARY KEY REFERENCES received_job(job_id),binding TEXT NOT NULL,bundle BLOB NOT NULL);
	CREATE TABLE IF NOT EXISTS expected_execution(job_id TEXT PRIMARY KEY REFERENCES pending_deployment(job_id),execution TEXT NOT NULL UNIQUE);
 CREATE TABLE IF NOT EXISTS deployment_recovery_hold(job_id TEXT PRIMARY KEY REFERENCES pending_deployment(job_id));
 UPDATE job_stage SET state='UNKNOWN' WHERE state='FAILED' AND stage_name='POST_VERIFY'
 AND EXISTS(SELECT 1 FROM execution_receipt r JOIN received_job j USING(job_id)
 WHERE r.job_id=job_stage.job_id AND r.terminal_state='UNKNOWN' AND j.state='UNKNOWN');`)
	if err != nil {
		a.Close()
		return nil, err
	}
	a.executor = helper
	return a, nil
}
func (a *Agent) admit(ctx context.Context, b policy.Bundle, s *Sample, job, binding string, refresh func() Sample) ([]byte, error) {
	// Lookups of admitted work need no external preflight. New work performs it
	// before acquiring a writer; the transaction below repeats the binding check.
	var knownBinding string
	readErr := a.db.QueryRowContext(ctx, "SELECT binding FROM pending_deployment WHERE job_id=?", job).Scan(&knownBinding)
	if readErr != nil && readErr != sql.ErrNoRows {
		return nil, readErr
	}
	start := time.Now()
	var approved policy.Authorized
	var preflightErr error
	if readErr == sql.ErrNoRows {
		approved, preflightErr = a.config.AuthorizeContext(ctx, b, s.UTC)
	}
	var checked time.Time
	if refresh != nil {
		*s = refresh()
		checked = s.UTC
	} else {
		checked = s.UTC.Add(time.Since(start))
	}
	lockStart := time.Now()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var known string
	var receipt []byte
	err = tx.QueryRow("SELECT p.binding,r.signed_receipt FROM pending_deployment p LEFT JOIN execution_receipt r USING(job_id) WHERE p.job_id=?", job).Scan(&known, &receipt)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	exists := err == nil
	if exists && binding != known {
		return nil, errors.New("IDEMPOTENCY_CONFLICT")
	}
	if len(receipt) > 0 {
		return receipt, nil
	}
	trusted, err := a.clock(tx, *s)
	if err != nil {
		return nil, err
	}
	if !trusted {
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return nil, errors.New("CLOCK_UNTRUSTED")
	}
	if exists {
		return nil, tx.Commit()
	}
	if preflightErr == nil && !approved.Current(checked.Add(time.Since(lockStart))) {
		preflightErr = errors.New("AUTHORITY_EXPIRED")
	}
	if preflightErr != nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return nil, preflightErr
	}
	var held int
	if err = tx.QueryRow("SELECT count(*) FROM deployment_recovery_hold").Scan(&held); err != nil {
		return nil, err
	}
	if held != 0 {
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return nil, errors.New("HISTORICAL_RECOVERY_REQUIRED")
	}
	if _, err = tx.Exec("SAVEPOINT reserve_job"); err != nil {
		return nil, err
	}
	if reserveErr := reserveJob(tx, approved, s.UTC.UTC().Format(time.RFC3339Nano), a.config.Installation, a.config.Scope.Site); reserveErr != nil {
		if _, err = tx.Exec("ROLLBACK TO reserve_job"); err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return nil, reserveErr
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec("INSERT INTO pending_deployment VALUES(?,?,?)", job, binding, raw); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("INSERT INTO expected_execution VALUES(?,?)", job, deployment.ExecutionID(binding)); err != nil {
		return nil, err
	}
	return nil, tx.Commit()
}
func (a *Agent) dispatch(ctx context.Context, b policy.Bundle, s Sample, refresh func() Sample) ([]byte, error) {
	binding, err := deployment.BundleDigest(b)
	if err != nil {
		return nil, err
	}
	var job contracts.MC03
	if err = json.Unmarshal(b.Job, &job); err != nil {
		return nil, err
	}
	id := job.Body.JobId
	receipt, err := a.admit(ctx, b, &s, id, binding, refresh)
	if err != nil || len(receipt) > 0 {
		return receipt, err
	}
	return a.deliver(ctx, b, s, job, binding, "lookup")
}

// resume is reachable only from the local pending queue. The canonical hashes
// commit to the originally verified signatures, product, artifact and signer.
// A retired issuer can never turn this read/fence path into an admission.
func (a *Agent) resume(ctx context.Context, b policy.Bundle, s Sample) ([]byte, error) {
	var job contracts.MC03
	if err := json.Unmarshal(b.Job, &job); err != nil {
		return nil, err
	}
	j := job.Body
	binding, err := deployment.BundleDigest(b)
	if err != nil {
		return nil, err
	}
	digest, err := policy.Digest(b.Job)
	if err != nil {
		return nil, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var n int
	err = tx.QueryRow(`SELECT count(*) FROM pending_deployment p JOIN received_job j USING(job_id)
 JOIN local_identity i USING(installation_id) WHERE j.job_id=? AND p.binding=? AND j.payload_sha256=? AND j.issuer_kid=?
 AND j.command_type='DEPLOY_RELEASE' AND j.verified_at IS NOT NULL AND j.state='RUNNING'
 AND j.installation_id=? AND j.site_id=? AND i.tenant_id=? AND i.agent_id=?`, j.JobId, binding, digest, job.Signature.Kid, j.InstallationId, job.SiteId, job.TenantId, j.AgentId).Scan(&n)
	if err != nil {
		return nil, err
	}
	if n != 1 || len(b.Grants) == 0 || len(j.ApprovalGrantIds) != len(b.Grants) {
		return nil, errors.New("HISTORICAL_ADMISSION_REQUIRED")
	}
	for _, raw := range b.Grants {
		var g contracts.MC09
		if err = json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		hash, e := policy.Digest(raw)
		if e != nil {
			return nil, e
		}
		if err = tx.QueryRow("SELECT count(*) FROM approval_consumption WHERE job_id=? AND grant_id=? AND grant_sha256=?", j.JobId, g.Body.GrantId, hash).Scan(&n); err != nil || n != 1 {
			return nil, errors.New("HISTORICAL_APPROVAL_REQUIRED")
		}
	}
	trusted, err := a.clock(tx, s)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if !trusted {
		return nil, errors.New("CLOCK_UNTRUSTED")
	}
	operation := "lookup"
	if _, err = signing.Verify("MC-03", b.Job, a.config.Scope, a.config.Trust); err != nil {
		operation = "recover"
	}
	wire, err := a.deliver(ctx, b, s, job, binding, operation)
	if err != nil && ctx.Err() == nil {
		// A late concurrent recovery may already have committed its receipt.
		_, holdErr := a.db.ExecContext(ctx, `INSERT OR IGNORE INTO deployment_recovery_hold
 SELECT job_id FROM received_job WHERE job_id=? AND state='RUNNING'`, j.JobId)
		if holdErr != nil {
			return nil, errors.Join(err, holdErr)
		}
	}
	return wire, err
}

func (a *Agent) deliver(ctx context.Context, b policy.Bundle, s Sample, job contracts.MC03, binding, operation string) ([]byte, error) {
	id := job.Body.JobId
	var receipt []byte
	reply, err := a.executor.Exchange(ctx, deployment.Request{Operation: operation, Bundle: b})
	if err != nil {
		return nil, err
	}
	if operation == "recover" && reply.Status != deployment.Done {
		return nil, errors.New("HISTORICAL_RECOVERY_REQUIRED")
	}
	if reply.Status == deployment.Absent || reply.Status == deployment.Pending {
		operation := "execute"
		if _, err = a.config.AuthorizeContext(ctx, b, s.UTC); err != nil {
			operation = "abort"
		}
		reply, err = a.executor.Exchange(ctx, deployment.Request{Operation: operation, Bundle: b})
		if err != nil {
			return nil, err
		}
	}
	if reply.Status == deployment.Refused {
		var params contracts.DeployReleaseParams
		if err = json.Unmarshal(job.Body.Params, &params); err != nil {
			return nil, err
		}
		digest, err := policy.Digest(b.Job)
		if err != nil {
			return nil, err
		}
		reply.Result = &deployment.Result{Job: id, JobDigest: digest, Artifact: params.ManifestSha256, Product: params.ProductKey, Execution: deployment.ExecutionID(binding), State: "REJECTED", Stage: "BEFORE_EFFECT", Code: "PRECONDITION_FAILED"}
	} else if reply.Status != deployment.Done || reply.Result == nil {
		return nil, errors.New("HELPER_UNAVAILABLE")
	}
	if err = reply.Result.Validate(b); err != nil {
		return nil, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// A pre-upgrade pending bundle can still reconcile its original random ID
	// through the pinned helper. New admissions pin the ID before dispatch.
	var expected string
	err = tx.QueryRow("SELECT execution FROM expected_execution WHERE job_id=?", id).Scan(&expected)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if operation == "recover" && err == sql.ErrNoRows {
		return nil, errors.New("HISTORICAL_EXECUTION_PIN_REQUIRED")
	}
	if err == nil && reply.Result.Execution != expected {
		return nil, errors.New("EXECUTION_ID")
	}
	err = tx.QueryRow("SELECT signed_receipt FROM execution_receipt WHERE job_id=?", id).Scan(&receipt)
	if err == nil {
		return receipt, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	now := s.UTC.UTC().Format(time.RFC3339Nano)
	out := *reply.Result
	stage := "FINISHED"
	if out.State == "UNKNOWN" {
		stage = "UNKNOWN"
	} else if out.State != "SUCCEEDED" {
		stage = "FAILED"
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	hash, err := policy.Digest(raw)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec("INSERT INTO job_stage VALUES(?,0,?,?,?, ?,?)", id, out.Stage, now, now, stage, hash)
	if err != nil {
		return nil, err
	}
	receipt, err = a.finish(tx, id, now, out)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM deployment_recovery_hold WHERE job_id=?", id); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return receipt, nil
}
