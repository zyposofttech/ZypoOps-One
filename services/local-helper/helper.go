// Package helper is an unprivileged simulation; its only operation records a test deployment in SQLite.
package helper

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
	"zyposoft.local/operations/packages/local-policy"
	"zyposoft.local/operations/packages/local-store"
	contracts "zyposoft.local/operations/packages/management-contracts"
)

type Helper struct {
	db       *sql.DB
	cfg      policy.Config
	uid, gid uint32
	now      func() time.Time
}

func Open(path string, c policy.Config, uid, gid uint32, now func() time.Time) (*Helper, error) {
	if len(c.TestArtifacts) == 0 || now == nil || len(c.Trust) == 0 || c.Installation == "" || c.Agent == "" {
		return nil, errors.New("HELPER_CONFIG")
	}
	db, err := localstore.Open(path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS simulated_deployment(job TEXT PRIMARY KEY, token TEXT NOT NULL UNIQUE,nonce TEXT NOT NULL UNIQUE,sequence INTEGER NOT NULL UNIQUE,digest TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS consumed_grant(id TEXT PRIMARY KEY);
 CREATE TABLE IF NOT EXISTS local_scope(singleton INTEGER PRIMARY KEY CHECK(singleton=1),tenant TEXT,site TEXT,installation TEXT,agent TEXT);
 CREATE TABLE IF NOT EXISTS time_floor(singleton INTEGER PRIMARY KEY CHECK(singleton=1),utc INTEGER NOT NULL,blocked INTEGER NOT NULL);`)
	if err == nil {
		_, err = db.Exec("INSERT OR IGNORE INTO local_scope VALUES(1,?,?,?,?)", c.Scope.Tenant, c.Scope.Site, c.Installation, c.Agent)
	}
	if err == nil {
		var count int
		err = db.QueryRow("SELECT count(*) FROM local_scope WHERE tenant=? AND site=? AND installation=? AND agent=?", c.Scope.Tenant, c.Scope.Site, c.Installation, c.Agent).Scan(&count)
		if count != 1 {
			err = errors.New("HELPER_SCOPE")
		}
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Helper{db, c.Clone(), uid, gid, now}, nil
}
func (h *Helper) Close() error { return h.db.Close() }
func (h *Helper) apply(raw []byte, uid, gid uint32) string {
	if uid != h.uid || gid != h.gid {
		return "UNAUTHORISED_PEER"
	}
	data, err := contracts.Canonical(raw)
	if err != nil {
		return "INVALID_REQUEST"
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return "INVALID_REQUEST"
	}
	for k := range fields {
		if k != "job" && k != "manifest" && k != "grants" && k != "backup" {
			return "INVALID_REQUEST"
		}
	}
	var b policy.Bundle
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&b) != nil || len(b.Grants) > 8 {
		return "INVALID_REQUEST"
	}
	tx, err := h.db.Begin()
	if err != nil {
		return "STORE_UNAVAILABLE"
	}
	defer tx.Rollback()
	now := h.now().UTC()
	_, err = tx.Exec("INSERT INTO time_floor VALUES(1,?,0) ON CONFLICT(singleton) DO UPDATE SET blocked=(blocked OR excluded.utc<utc),utc=max(utc,excluded.utc)", now.UnixNano())
	if err != nil {
		return "STORE_UNAVAILABLE"
	}
	var blocked bool
	if err = tx.QueryRow("SELECT blocked FROM time_floor").Scan(&blocked); err != nil {
		return "STORE_UNAVAILABLE"
	}
	if blocked {
		if tx.Commit() != nil {
			return "STORE_UNAVAILABLE"
		}
		return "CLOCK_UNTRUSTED"
	}
	if _, err = tx.Exec("SAVEPOINT deployment"); err != nil {
		return "STORE_UNAVAILABLE"
	}
	code := h.simulate(tx, b, now)
	if code != "SIMULATED" {
		if _, err = tx.Exec("ROLLBACK TO deployment"); err != nil {
			return "STORE_UNAVAILABLE"
		}
	}
	if tx.Commit() != nil {
		return "STORE_UNAVAILABLE"
	}
	return code
}
func (h *Helper) simulate(tx *sql.Tx, b policy.Bundle, now time.Time) string {
	approved, err := h.cfg.Authorize(b, now)
	if err != nil || approved.Job.Body.CommandType != "DEPLOY_RELEASE" {
		return "OPERATION_DENIED"
	}
	j := approved.Job.Body
	var floor int64
	if err = tx.QueryRow("SELECT coalesce(max(sequence),0) FROM simulated_deployment").Scan(&floor); err != nil {
		return "STORE_UNAVAILABLE"
	}
	if j.Sequence <= floor {
		return "REPLAY_DENIED"
	}
	for grant := range approved.Grants {
		if _, err = tx.Exec("INSERT INTO consumed_grant VALUES(?)", grant); err != nil {
			return "REPLAY_DENIED"
		}
	}
	if _, err = tx.Exec("INSERT INTO simulated_deployment VALUES(?,?,?,?,?)", j.JobId, j.IdempotencyKey, j.Nonce, j.Sequence, approved.Digest); err != nil {
		return "REPLAY_DENIED"
	}
	return "SIMULATED"
}
