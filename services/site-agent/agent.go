// Package agent owns only technical state in the local SQLite ledger.
package agent

import (
	"context"
	"crypto"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"strings"
	"time"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-policy"
	"zyposoft.local/operations/packages/local-store"
	contracts "zyposoft.local/operations/packages/management-contracts"
	"zyposoft.local/operations/packages/signing-go"
)

//go:embed agent_sqlite_v1_2.sql
var migration string

type Identity struct{ Tenant, Site, Installation, Agent, HostHash string }

func LoadIdentity(raw []byte) (Identity, error) {
	var id Identity
	data, err := contracts.Canonical(raw)
	if err != nil {
		return id, err
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&id); err != nil {
		return id, err
	}
	for _, v := range []string{id.Tenant, id.Site, id.Installation, id.Agent} {
		if _, err := uuid.Parse(v); err != nil {
			return id, errors.New("INVALID_IDENTITY")
		}
	}
	if len(id.HostHash) != 64 || strings.Trim(id.HostHash, "0123456789abcdef") != "" {
		return id, errors.New("INVALID_HOST_BINDING")
	}
	return id, nil
}

type TestKeyProvider interface {
	ReceiptKey() (string, crypto.PrivateKey, error)
}
type Sample struct {
	UTC       time.Time
	Boot      string
	Monotonic int64
	Trusted   bool
}
type Agent struct {
	db       *sql.DB
	config   policy.Config
	keys     TestKeyProvider
	log      *slog.Logger
	fault    func(string) error
	recovery *TestTimeSource
	executor deployment.Helper
}

func Open(path string, id Identity, c policy.Config, keys TestKeyProvider, log *slog.Logger) (*Agent, error) {
	return OpenWithTestTimeSource(path, id, c, keys, log, nil)
}
func OpenWithTestTimeSource(path string, id Identity, c policy.Config, keys TestKeyProvider, log *slog.Logger, recovery *TestTimeSource) (*Agent, error) {
	encoded, _ := json.Marshal(id)
	if _, err := LoadIdentity(encoded); err != nil {
		return nil, err
	}
	if keys == nil || c.Scope.Tenant != id.Tenant || c.Scope.Site != id.Site || c.Installation != id.Installation || c.Agent != id.Agent {
		return nil, errors.New("CONFIG_SCOPE")
	}
	db, err := localstore.Open(path)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(migration); err != nil {
		db.Close()
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec("INSERT INTO local_identity(installation_id,tenant_id,site_id,agent_id,host_binding_sha256,created_at) SELECT ?,?,?,?,?,? WHERE NOT EXISTS(SELECT 1 FROM local_identity)", id.Installation, id.Tenant, id.Site, id.Agent, id.HostHash, time.Now().UTC().Format(time.RFC3339Nano))
	if err == nil {
		var count int
		err = tx.QueryRow("SELECT count(*) FROM local_identity WHERE installation_id=? AND tenant_id=? AND site_id=? AND agent_id=? AND host_binding_sha256=?", id.Installation, id.Tenant, id.Site, id.Agent, id.HostHash).Scan(&count)
		if err == nil && count != 1 {
			err = errors.New("IDENTITY_CHANGED")
		}
	}
	if err == nil {
		_, err = tx.Exec("INSERT OR IGNORE INTO agent_sequence(installation_id,last_sequence,updated_at) VALUES(?,0,?)", id.Installation, time.Now().UTC().Format(time.RFC3339Nano))
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, err
	}
	if log == nil {
		log = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return &Agent{db: db, config: c.Clone(), keys: keys, log: log, recovery: recovery}, nil
}
func (a *Agent) Close() error { return a.db.Close() }
func (a *Agent) clock(tx *sql.Tx, s Sample) (bool, error) {
	var last, boot, confidence string
	var monotonic int64
	err := tx.QueryRow("SELECT last_trusted_utc,last_monotonic_ns,boot_id,clock_confidence FROM clock_watermark WHERE installation_id=?", a.config.Installation).Scan(&last, &monotonic, &boot, &confidence)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	now := s.UTC.UTC().Format(time.RFC3339Nano)
	trusted := s.Trusted && s.Boot != "" && s.Monotonic >= 0 && !s.UTC.IsZero()
	if err == sql.ErrNoRows && !trusted {
		// Only a trusted bootstrap sample can establish the first UTC floor.
		return false, nil
	}
	if err == nil {
		floor, e := time.Parse(time.RFC3339Nano, last)
		if e != nil {
			return false, e
		}
		trusted = trusted && confidence == "TRUSTED" && !s.UTC.Before(floor) && boot == s.Boot && s.Monotonic >= monotonic
		if !trusted {
			now = last
			if s.Boot == "" || s.Monotonic < 0 {
				s.Boot = boot
				s.Monotonic = monotonic
			}
			if s.Boot == boot {
				s.Monotonic = max(monotonic, s.Monotonic)
			}
		}
	}
	confidence = "UNTRUSTED"
	if trusted {
		confidence = "TRUSTED"
	}
	_, err = tx.Exec("INSERT INTO clock_watermark VALUES(?,?,?,?,?) ON CONFLICT(installation_id) DO UPDATE SET last_trusted_utc=excluded.last_trusted_utc,last_monotonic_ns=excluded.last_monotonic_ns,boot_id=excluded.boot_id,clock_confidence=excluded.clock_confidence", a.config.Installation, now, max(s.Monotonic, 0), s.Boot, confidence)
	return trusted, err
}
func (a *Agent) Handle(ctx context.Context, b policy.Bundle, s Sample) ([]byte, error) {
	// An invalid network message cannot poison the trusted-clock watermark.
	doc, err := signing.Verify("MC-03", b.Job, a.config.Scope, a.config.Trust)
	if err != nil {
		return nil, err
	}
	if a.executor != nil && doc["body"].(signing.Object)["command_type"] == "DEPLOY_RELEASE" {
		return a.dispatch(ctx, b, s)
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
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
	// Preserve a trusted clock observation even when the job is rejected. A job's
	// sequence, grants, simulated effect and receipt still succeed or roll back together.
	if _, err = tx.Exec("SAVEPOINT job_processing"); err != nil {
		return nil, err
	}
	wire, jobErr := a.process(tx, b, s)
	if jobErr != nil {
		if _, err = tx.Exec("ROLLBACK TO job_processing"); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if jobErr == nil {
		a.log.Info("job_receipt_ready", "stage", "SIMULATION")
	}
	return wire, jobErr
}

func (a *Agent) process(tx *sql.Tx, b policy.Bundle, s Sample) ([]byte, error) {
	authorized, err := a.config.Authorize(b, s.UTC)
	if err != nil {
		return nil, err
	}
	j := authorized.Job.Body
	now := s.UTC.UTC().Format(time.RFC3339Nano)
	var existing, digest string
	err = tx.QueryRow("SELECT r.signed_receipt,j.payload_sha256 FROM execution_receipt r JOIN received_job j USING(job_id) WHERE job_id=?", j.JobId).Scan(&existing, &digest)
	if err == nil {
		if digest != authorized.Digest {
			return nil, errors.New("IDEMPOTENCY_CONFLICT")
		}
		return []byte(existing), nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	if err = reserveJob(tx, authorized, now, a.config.Installation, a.config.Scope.Site); err != nil {
		return nil, err
	}
	// The only effect is this database row. It commits atomically with the receipt.
	if _, err = tx.Exec("INSERT INTO job_stage VALUES(?,0,'SIMULATION',?,?,'FINISHED',NULL)", j.JobId, now, now); err != nil {
		return nil, err
	}
	if a.fault != nil {
		if err = a.fault("before_receipt"); err != nil {
			return nil, err
		}
	}
	return a.finish(tx, j.JobId, now, deployment.Result{Execution: uuid.NewString(), State: "SUCCEEDED", Stage: "SIMULATION", Artifact: authorized.Digest, Code: "NONE"})
}
func reserveJob(tx *sql.Tx, authorized policy.Authorized, now, installation, site string) error {
	j := authorized.Job.Body
	res, err := tx.Exec("UPDATE agent_sequence SET last_sequence=?,updated_at=? WHERE installation_id=? AND last_sequence=?", j.Sequence, now, installation, j.Sequence-1)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil || count != 1 {
		return errors.New("SEQUENCE_REJECTED")
	}
	_, err = tx.Exec("INSERT INTO received_job(job_id,installation_id,site_id,command_type,idempotency_key,payload_sha256,nonce,issuer_kid,sequence,not_before,expires_at,state,received_at,verified_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,'RECEIVED',?,?)", j.JobId, j.InstallationId, site, j.CommandType, j.IdempotencyKey, authorized.Digest, j.Nonce, authorized.Job.Signature.Kid, j.Sequence, j.NotBefore, j.ExpiresAt, now, now)
	if err != nil {
		return errors.New("IDEMPOTENCY_CONFLICT")
	}
	for id, hash := range authorized.Grants {
		if _, err = tx.Exec("INSERT INTO approval_consumption VALUES(?,?,?,?)", id, j.JobId, hash, now); err != nil {
			return errors.New("GRANT_CONSUMED")
		}
	}
	for _, state := range []string{"RECEIVED", "VALIDATED", "ACCEPTED", "RUNNING"} {
		if _, err = tx.Exec("UPDATE received_job SET state=? WHERE job_id=?", state, j.JobId); err != nil {
			return err
		}
		if _, err = tx.Exec("INSERT INTO job_progress_event VALUES(?,?,?,?,?)", uuid.NewString(), j.JobId, state, now, "MOCK"); err != nil {
			return err
		}
	}
	return nil
}
func (a *Agent) finish(tx *sql.Tx, job, now string, out deployment.Result) ([]byte, error) {
	receipt := uuid.NewString()
	doc := signing.Object{"contract_id": "MC-04", "schema_version": "1.2.0", "message_id": receipt, "site_id": a.config.Scope.Site, "tenant_id": a.config.Scope.Tenant, "sent_at": now, "body": signing.Object{"job_id": job, "execution_id": out.Execution, "state": out.State, "stage": out.Stage, "observed_sha256": out.Artifact, "error_code": out.Code, "event_at": now, "clock_confidence": "TRUSTED"}}
	unsigned, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	kid, priv, err := a.keys.ReceiptKey()
	if err != nil {
		return nil, err
	}
	wire, err := signing.Sign("MC-04", unsigned, a.config.Scope, a.config.Trust, kid, priv)
	if err != nil {
		return nil, err
	}
	hash, err := policy.Digest(wire)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec("INSERT INTO execution_receipt VALUES(?,?,?,?,?,?,?,NULL)", receipt, job, out.Execution, out.State, hash, string(wire), now); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("UPDATE received_job SET state=? WHERE job_id=?", out.State, job); err != nil {
		return nil, err
	}
	return wire, nil
}

type Transport interface {
	Next(context.Context) (policy.Bundle, error)
	Receipt(context.Context, []byte) error
}

// Run is outbound only. Pending receipts are sent on startup and before fetching new work.
func (a *Agent) Run(ctx context.Context, t Transport, clock func() Sample) error {
	for ctx.Err() == nil {
		var id, wire string
		err := a.db.QueryRowContext(ctx, "SELECT receipt_id,signed_receipt FROM execution_receipt WHERE acknowledged_at IS NULL ORDER BY completed_at,receipt_id LIMIT 1").Scan(&id, &wire)
		if err == nil {
			if err = t.Receipt(ctx, []byte(wire)); err != nil {
				return fmt.Errorf("RECEIPT_OFFLINE")
			}
			if _, err = a.db.ExecContext(ctx, "UPDATE execution_receipt SET acknowledged_at=? WHERE receipt_id=?", clock().UTC.UTC().Format(time.RFC3339Nano), id); err != nil {
				return err
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if a.executor != nil {
			var pending []byte
			err = a.db.QueryRowContext(ctx, "SELECT bundle FROM pending_deployment p JOIN received_job j USING(job_id) WHERE j.state='RUNNING' ORDER BY j.sequence LIMIT 1").Scan(&pending)
			if err == nil {
				var b policy.Bundle
				if err = json.Unmarshal(pending, &b); err != nil {
					return err
				}
				if _, err = a.Handle(ctx, b, clock()); err != nil {
					return err
				}
				continue
			}
			if err != sql.ErrNoRows {
				return err
			}
		}
		b, err := t.Next(ctx)
		if err != nil {
			return err
		}
		if _, err = a.Handle(ctx, b, clock()); err != nil {
			a.log.Warn("job_rejected", "code", "MANAGEMENT_REJECTED")
			if a.executor != nil {
				return err
			}
			continue
		}
	}
	return ctx.Err()
}
