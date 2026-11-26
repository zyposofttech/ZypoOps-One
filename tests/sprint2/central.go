// Package sprint2 provides an isolated, synthetic central dispatcher for the integration harness.
package sprint2

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"maps"
	"net/http"
	"time"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-policy"
	"zyposoft.local/operations/packages/local-store"
	contracts "zyposoft.local/operations/packages/management-contracts"
	"zyposoft.local/operations/packages/signing-go"
)

// Enrolment is supplied by the local test operator, independently of jobs and receipts.
type Enrolment struct{ Tenant, Site, Installation, Agent, ReceiptKey string }

type Central struct {
	db    *sql.DB
	scope signing.Scope
	trust signing.Trust
}

func OpenCentral(path string, scope signing.Scope, trust signing.Trust, provision ...Enrolment) (*Central, error) {
	db, err := localstore.Open(path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`PRAGMA journal_mode=WAL;CREATE TABLE IF NOT EXISTS commands(job TEXT PRIMARY KEY,seq INTEGER NOT NULL UNIQUE,binding TEXT NOT NULL,artifact TEXT NOT NULL,bundle BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS terminal(job TEXT PRIMARY KEY REFERENCES commands(job),execution TEXT UNIQUE NOT NULL,binding TEXT NOT NULL,receipt BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS command_execution(job TEXT PRIMARY KEY REFERENCES commands(job),execution TEXT NOT NULL UNIQUE);
 CREATE TABLE IF NOT EXISTS enrolment(tenant TEXT NOT NULL,site TEXT NOT NULL,installation TEXT NOT NULL UNIQUE,agent TEXT NOT NULL UNIQUE,receipt_key TEXT NOT NULL UNIQUE,PRIMARY KEY(tenant,site,installation,agent));
 CREATE UNIQUE INDEX IF NOT EXISTS command_target_sequence ON commands(json_extract(CAST(bundle AS TEXT),'$.job.body.installation_id'),json_extract(CAST(bundle AS TEXT),'$.job.body.sequence'));`)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = configureEnrolments(db, scope, trust, provision); err != nil {
		db.Close()
		return nil, err
	}
	return &Central{db, scope, maps.Clone(trust)}, nil
}

func configureEnrolments(db *sql.DB, scope signing.Scope, trust signing.Trust, records []Enrolment) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, r := range records {
		key, ok := trust[r.ReceiptKey]
		if !ok || key.Revoked || key.Purpose != "agent_receipt" || key.Scope != (signing.Scope{Tenant: r.Tenant, Site: r.Site}) {
			return errors.New("ENROLMENT_KEY")
		}
		for _, id := range []string{r.Tenant, r.Site, r.Installation, r.Agent} {
			if _, e := uuid.Parse(id); e != nil {
				return errors.New("ENROLMENT_ID")
			}
		}
		_, err = tx.Exec("INSERT INTO enrolment VALUES(?,?,?,?,?)", r.Tenant, r.Site, r.Installation, r.Agent, r.ReceiptKey)
		if err != nil {
			return err
		}
	}
	var n int
	if err = tx.QueryRow("SELECT count(*) FROM enrolment WHERE tenant=? AND site=?", scope.Tenant, scope.Site).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return errors.New("CENTRAL_SCOPE")
	}
	return tx.Commit()
}

func (c *Central) Close() error { return c.db.Close() }
func (c *Central) Submit(b policy.Bundle) error {
	if _, err := signing.Verify("MC-03", b.Job, c.scope, c.trust); err != nil {
		return err
	}
	raw, err := json.Marshal(b)
	if err != nil || len(raw) > contracts.MaxBytes {
		return errors.New("COMMAND_SIZE")
	}
	binding, err := deployment.BundleDigest(b)
	if err != nil {
		return err
	}
	var job contracts.MC03
	var params contracts.DeployReleaseParams
	if json.Unmarshal(b.Job, &job) != nil || job.Body.CommandType != "DEPLOY_RELEASE" || json.Unmarshal(job.Body.Params, &params) != nil {
		return errors.New("INVALID_JOB")
	}
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var enrolled int
	if err = tx.QueryRow("SELECT count(*) FROM enrolment WHERE tenant=? AND site=? AND installation=? AND agent=?", c.scope.Tenant, c.scope.Site, job.Body.InstallationId, job.Body.AgentId).Scan(&enrolled); err != nil {
		return err
	}
	if enrolled != 1 {
		return errors.New("INVALID_JOB")
	}
	created, err := tx.Exec("INSERT INTO commands VALUES(?,(SELECT coalesce(max(seq),0)+1 FROM commands),?,?,?) ON CONFLICT(job) DO NOTHING", job.Body.JobId, binding, params.ManifestSha256, raw)
	if err != nil {
		return err
	}
	var previous string
	if err = tx.QueryRow("SELECT binding FROM commands WHERE job=?", job.Body.JobId).Scan(&previous); err != nil || previous != binding {
		return errors.New("COMMAND_CONFLICT")
	}
	n, err := created.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		if _, err = tx.Exec("INSERT INTO command_execution VALUES(?,?)", job.Body.JobId, deployment.ExecutionID(binding)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// restoreLegacyReservation provisions a pre-upgrade random ID from a trusted
// helper-ledger snapshot. It is local to the simulator and has no HTTP route.
func (c *Central) restoreLegacyReservation(job, binding, execution string) error {
	if _, err := uuid.Parse(execution); err != nil {
		return err
	}
	res, err := c.db.Exec(`INSERT INTO command_execution SELECT job,? FROM commands
		WHERE job=? AND binding=? AND NOT EXISTS(SELECT 1 FROM terminal WHERE job=commands.job)`, execution, job, binding)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("LEGACY_BINDING")
	}
	return nil
}
func (c *Central) accept(wire []byte) error {
	if _, err := signing.Verify("MC-04", wire, c.scope, c.trust); err != nil {
		return err
	}
	var r contracts.MC04
	if err := json.Unmarshal(wire, &r); err != nil {
		return err
	}
	binding, err := policy.Digest(wire)
	if err != nil {
		return err
	}
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var artifact, expected, signer string
	err = tx.QueryRow(`SELECT artifact,coalesce(execution,''),receipt_key FROM commands LEFT JOIN command_execution USING(job)
 JOIN enrolment e ON e.tenant=json_extract(CAST(bundle AS TEXT),'$.job.tenant_id')
 AND e.site=json_extract(CAST(bundle AS TEXT),'$.job.site_id')
 AND e.installation=json_extract(CAST(bundle AS TEXT),'$.job.body.installation_id')
 AND e.agent=json_extract(CAST(bundle AS TEXT),'$.job.body.agent_id')
 WHERE job=? AND e.tenant=? AND e.site=?`, r.Body.JobId, c.scope.Tenant, c.scope.Site).Scan(&artifact, &expected, &signer)
	if err != nil || artifact != r.Body.ObservedSha256 || r.Signature.Kid != signer {
		return errors.New("UNKNOWN_OUTCOME")
	}
	var original string
	err = tx.QueryRow("SELECT binding FROM terminal WHERE job=?", r.Body.JobId).Scan(&original)
	if err == nil {
		if original != binding {
			return errors.New("OUTCOME_CONFLICT")
		}
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	// MC-04 has no installation/agent fields. The test enrolment pins this
	// receipt signer to the installation/agent pair accepted at submission.
	outcome := deployment.Result{Stage: r.Body.Stage, State: r.Body.State, Code: r.Body.ErrorCode}
	if r.Body.ClockConfidence != "TRUSTED" ||
		r.Body.ExecutionId != expected || outcome.ValidateOutcome() != nil {
		return errors.New("RECEIPT_BINDING")
	}
	if _, err = tx.Exec("INSERT INTO terminal VALUES(?,?,?,?)", r.Body.JobId, r.Body.ExecutionId, binding, wire); err != nil {
		return err
	}
	return tx.Commit()
}
func (c *Central) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET" && r.URL.Path == "/next":
		var raw []byte
		err := c.db.QueryRowContext(r.Context(), "SELECT bundle FROM commands LEFT JOIN terminal USING(job) WHERE terminal.job IS NULL AND json_extract(CAST(bundle AS TEXT),'$.job.tenant_id')=? AND json_extract(CAST(bundle AS TEXT),'$.job.site_id')=? ORDER BY seq LIMIT 1", c.scope.Tenant, c.scope.Site).Scan(&raw)
		if err == sql.ErrNoRows {
			w.WriteHeader(204)
			return
		}
		if err != nil {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write(raw)
	case r.Method == "POST" && r.URL.Path == "/receipt":
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, contracts.MaxBytes))
		if err != nil || c.accept(data) != nil {
			w.WriteHeader(409)
			return
		}
		w.WriteHeader(204)
	default:
		w.WriteHeader(404)
	}
}

type Outbound struct{ LoopbackURL string }

func (o Outbound) call(ctx context.Context, method, path string, data []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, o.LoopbackURL+path, bytes.NewReader(data))
	if err != nil {
		return 0, nil, err
	}
	cli := http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := cli.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	wire, err := io.ReadAll(io.LimitReader(resp.Body, contracts.MaxBytes+1))
	if len(wire) > contracts.MaxBytes {
		return 0, nil, errors.New("PAYLOAD_LIMIT")
	}
	return resp.StatusCode, wire, err
}
func (o Outbound) Next(ctx context.Context) (policy.Bundle, error) {
	var b policy.Bundle
	status, data, err := o.call(ctx, "GET", "/next", nil)
	if err != nil {
		return b, err
	}
	if status == 204 {
		return b, io.EOF
	}
	if status != 200 {
		return b, errors.New("OFFLINE")
	}
	raw, err := contracts.Canonical(data)
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(raw, &b)
	return b, err
}
func (o Outbound) Receipt(ctx context.Context, data []byte) error {
	status, _, err := o.call(ctx, "POST", "/receipt", data)
	if err != nil || status != 204 {
		return errors.New("ACK_MISSING")
	}
	return nil
}
