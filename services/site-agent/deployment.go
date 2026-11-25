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
	CREATE TABLE IF NOT EXISTS expected_execution(job_id TEXT PRIMARY KEY REFERENCES pending_deployment(job_id),execution TEXT NOT NULL UNIQUE)`)
	if err != nil {
		a.Close()
		return nil, err
	}
	a.executor = helper
	return a, nil
}
func (a *Agent) admit(ctx context.Context, b policy.Bundle, s Sample, job, binding string) ([]byte, error) {
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
	trusted, err := a.clock(tx, s)
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
	approved, err := a.config.AuthorizeContext(ctx, b, s.UTC)
	if err != nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return nil, err
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
func (a *Agent) dispatch(ctx context.Context, b policy.Bundle, s Sample) ([]byte, error) {
	binding, err := deployment.BundleDigest(b)
	if err != nil {
		return nil, err
	}
	var job contracts.MC03
	if err = json.Unmarshal(b.Job, &job); err != nil {
		return nil, err
	}
	id := job.Body.JobId
	receipt, err := a.admit(ctx, b, s, id, binding)
	if err != nil || len(receipt) > 0 {
		return receipt, err
	}
	reply, err := a.executor.Exchange(ctx, deployment.Request{Operation: "lookup", Bundle: b})
	if err != nil {
		return nil, err
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
	if out.State != "SUCCEEDED" {
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
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return receipt, nil
}
