//go:build linux

// Package simulator runs isolated synthetic applications. Its SQLite file holds only test release and activity state.
package simulator

import (
	"bytes"
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"
	"zyposoft.local/operations/packages/local-store"
	contracts "zyposoft.local/operations/packages/management-contracts"
	pl "zyposoft.local/operations/packages/product-local"
	"zyposoft.local/operations/packages/unixpeer"
)

type Profile struct{ Product, Lock, Healthy, Initial, Artifact, Release string }
type App struct {
	db      *sql.DB
	profile Profile
	token   string
	now     func() time.Time
}

func Open(path string, p Profile, token string, now func() time.Time) (*App, error) {
	if p.Product == "" || p.Artifact == "" || p.Release == "" || len(token) < 32 || now == nil {
		return nil, errors.New("PRODUCT_CONFIG")
	}
	db, err := localstore.Open(path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS application(id INTEGER PRIMARY KEY CHECK(id=1),product TEXT NOT NULL,version TEXT NOT NULL,revision INTEGER NOT NULL,active INTEGER NOT NULL,ready INTEGER NOT NULL,mode TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS no_effect(job TEXT PRIMARY KEY,execution TEXT UNIQUE,artifact TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS effect(job TEXT PRIMARY KEY,execution TEXT UNIQUE,artifact TEXT NOT NULL,phase TEXT NOT NULL);`)
	if err == nil {
		_, err = db.Exec("INSERT OR IGNORE INTO application VALUES(1,?,?,1,0,1,'')", p.Product, p.Initial)
	}
	var product string
	if err == nil {
		err = db.QueryRow("SELECT product FROM application").Scan(&product)
		if err == nil && product != p.Product {
			err = errors.New("PRODUCT_BINDING")
		}
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return &App{db, p, token, now}, nil
}
func (a *App) Close() error { return a.db.Close() }

// SetActivity represents a local application event. It serializes with the effect's final safety check.
func (a *App) SetActivity(active, ready bool, mode string) error {
	_, err := a.db.Exec("UPDATE application SET active=?,ready=?,mode=?,revision=revision+1 WHERE id=1", active, ready, mode)
	return err
}
func (a *App) Serve(ctx context.Context, path string, peer unixpeer.Peer) error {
	l, err := unixpeer.Listen(path)
	if err != nil {
		return err
	}
	server := http.Server{Handler: a, ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, MaxHeaderBytes: 4096}
	stop := context.AfterFunc(ctx, func() { server.Close() })
	defer stop()
	err = server.Serve(unixpeer.Listener{OwnedListener: l, Allowed: peer})
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var active, ready bool
	var revision, version, mode string
	if err := a.db.QueryRowContext(r.Context(), "SELECT active,ready,revision,version,mode FROM application").Scan(&active, &ready, &revision, &version, &mode); err != nil {
		w.WriteHeader(503)
		return
	}
	var response any
	switch {
	case r.Method == "GET" && r.URL.Path == "/local/v1/locks":
		if mode == "TIMEOUT" {
			<-r.Context().Done()
			return
		}
		if mode == "MALFORMED" {
			response = map[string]string{"wrong": "field"}
			break
		}
		reasons := []string{}
		if active {
			reasons = append(reasons, a.profile.Lock)
		}
		response = contracts.LocksResponse{Locked: active, Reasons: reasons, LeaseTtlS: 1, Revision: revision}
	case r.Method == "GET" && r.URL.Path == "/local/v1/readiness":
		if mode == "POST_VERIFY_OFFLINE" && version == a.profile.Release {
			w.WriteHeader(503)
			return
		}
		code := a.profile.Healthy
		if !ready {
			code = "UNREADY"
		}
		response = contracts.ReadinessResponse{Ready: ready, ServiceCodes: []string{code}, Version: version}
	case r.Method == "POST" && (r.URL.Path == "/simulation/apply" || r.URL.Path == "/simulation/lookup" || r.URL.Path == "/simulation/fence"):
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+a.token)) != 1 {
			w.WriteHeader(403)
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2048))
		if err != nil {
			w.WriteHeader(400)
			return
		}
		raw, err = contracts.Canonical(raw)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		var cmd pl.Activation
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&cmd) != nil || cmd.Product != a.profile.Product || cmd.Artifact != a.profile.Artifact || cmd.Release != a.profile.Release {
			w.WriteHeader(403)
			return
		}
		if !validID(cmd.Job) || !validID(cmd.Execution) {
			w.WriteHeader(400)
			return
		}
		result, err := a.transition(r.Context(), cmd, r.URL.Path)
		if err != nil {
			w.WriteHeader(409)
			return
		}
		response = result
	default:
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
func validID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}
func (a *App) transition(ctx context.Context, c pl.Activation, operation string) (pl.Observation, error) {
	o := pl.Observation{Activation: c, Phase: "ABSENT"}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return o, err
	}
	defer tx.Rollback()
	var job, execution, artifact string
	err = tx.QueryRow(`SELECT job,execution,artifact,phase FROM effect WHERE job=?1 OR execution=?2
		UNION ALL SELECT job,execution,artifact,'BLOCKED' FROM no_effect WHERE job=?1 OR execution=?2`, c.Job, c.Execution).Scan(&job, &execution, &artifact, &o.Phase)
	if err == nil {
		if job != c.Job || execution != c.Execution || artifact != c.Artifact {
			return o, errors.New("CONFLICT")
		}
		return o, nil
	}
	if err != sql.ErrNoRows {
		return o, err
	}
	if operation == "/simulation/lookup" {
		return o, nil
	}
	var active, ready bool
	var mode string
	if err = tx.QueryRow("SELECT active,ready,mode FROM application").Scan(&active, &ready, &mode); err != nil {
		return o, err
	}
	expiry, err := time.Parse(time.RFC3339Nano, c.Expires)
	if operation == "/simulation/fence" || err != nil || !a.now().Before(expiry) || active || !ready {
		o.Phase = "BLOCKED"
		if _, err = tx.Exec("INSERT INTO no_effect VALUES(?,?,?)", c.Job, c.Execution, c.Artifact); err != nil {
			return o, err
		}
		return o, tx.Commit()
	}
	// The final lock check and the sole simulated release mutation use the same immediate transaction.
	o.Phase = "APPLIED"
	if mode == "AFTER_EFFECT" {
		o.Phase = "APPLIED_ERROR"
	}
	_, err = tx.Exec("INSERT INTO effect VALUES(?,?,?,?)", c.Job, c.Execution, c.Artifact, o.Phase)
	if err != nil {
		return o, err
	}
	_, err = tx.Exec("UPDATE application SET version=?,revision=revision+1,ready=? WHERE id=1", c.Release, mode != "POST_UNREADY")
	if err != nil {
		return o, err
	}
	return o, tx.Commit()
}
