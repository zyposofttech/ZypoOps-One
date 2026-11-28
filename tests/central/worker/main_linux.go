// Command worker runs only synthetic product profiles over private inherited pipes.
package main

import (
	"bufio"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"zyposoft.local/operations/packages/deployment"
	policy "zyposoft.local/operations/packages/local-policy"
	store "zyposoft.local/operations/packages/local-store"
	pl "zyposoft.local/operations/packages/product-local"
	signing "zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/packages/unixpeer"
	secondary "zyposoft.local/operations/products/mock-secondary"
	"zyposoft.local/operations/products/simulator"
	sono "zyposoft.local/operations/products/zyposono-sim"
	helper "zyposoft.local/operations/services/local-helper"
	agent "zyposoft.local/operations/services/site-agent"
)

type setup struct {
	Identity                                                                                  agent.Identity
	Product, Artifact, Job, BackupDigest, Category, BackupKid, ReceiptKid, ProgressKid, Token string
	ReceiptPrivate, ProgressPrivate                                                           []byte
	Trust                                                                                     []struct {
		Kid, Purpose, Alg, Tenant, Site string
		Public                          []byte
	}
}
type keys struct {
	kid string
	key crypto.PrivateKey
}

func (k keys) ReceiptKey() (string, crypto.PrivateKey, error) { return k.kid, k.key, nil }

type pipe struct {
	progressLedger *sql.DB
	in             *bufio.Scanner
	out            *json.Encoder
	cfg            setup
	policy         policy.Config
	progress       crypto.PrivateKey
}
type reply struct {
	Status string
	Bundle policy.Bundle
}

func (p *pipe) call(op string, wire json.RawMessage) (reply, error) {
	var r reply
	if e := p.out.Encode(struct {
		Op, Token string
		Wire      json.RawMessage
	}{op, p.cfg.Token, wire}); e != nil {
		return r, e
	}
	if !p.in.Scan() {
		return r, io.ErrUnexpectedEOF
	}
	if e := json.Unmarshal(p.in.Bytes(), &r); e != nil {
		return r, e
	}
	if r.Status == "empty" {
		return r, io.EOF
	}
	if r.Status != "ok" {
		return r, errors.New("CENTRAL_OFFLINE")
	}
	return r, nil
}
func (p *pipe) Next(ctx context.Context) (policy.Bundle, error) {
	r, e := p.call("next", nil)
	if e != nil {
		return r.Bundle, e
	}
	job, e := signing.Verify("MC-03", r.Bundle.Job, p.policy.Scope, p.policy.Trust)
	if e != nil {
		return r.Bundle, e
	}
	binding, e := deployment.BundleDigest(r.Bundle)
	if e != nil {
		return r.Bundle, e
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	body := map[string]any{"job_id": job["body"].(map[string]any)["job_id"], "execution_id": deployment.ExecutionID(binding), "stage_code": "VERIFY_ARTIFACT", "progress_state": "RECEIVED", "event_at": now, "sequence_number": 1}
	doc, e := json.Marshal(map[string]any{"contract_id": "AUX-01", "schema_version": "1.2.0", "message_id": uuid.NewString(), "tenant_id": p.cfg.Identity.Tenant, "site_id": p.cfg.Identity.Site, "sent_at": now, "body": body})
	if e != nil {
		return r.Bundle, e
	}
	msg, e := signing.Sign("AUX-01", doc, p.policy.Scope, p.policy.Trust, p.cfg.ProgressKid, p.progress)
	if e != nil {
		return r.Bundle, e
	}
	// Preserve the first signed observation across a failed progress acknowledgement.
	if _, e = p.progressLedger.Exec("INSERT INTO event(binding,wire) VALUES(?,?) ON CONFLICT(binding) DO NOTHING", binding, msg); e != nil {
		return r.Bundle, e
	}
	if e = p.progressLedger.QueryRow("SELECT wire FROM event WHERE binding=?", binding).Scan(&msg); e != nil {
		return r.Bundle, e
	}
	_, e = p.call("progress", msg)
	return r.Bundle, e
}
func (p *pipe) Receipt(ctx context.Context, wire []byte) error {
	_, e := p.call("receipt", wire)
	return e
}

// The Agent reaches Exchange only after its SQLite admission has committed.
type pendingWitness struct {
	socket    deployment.Socket
	transport *pipe
	ledger    string
	observed  bool
}

func (w *pendingWitness) Exchange(ctx context.Context, req deployment.Request) (deployment.Reply, error) {
	if !w.observed {
		db, e := store.Open(w.ledger)
		if e != nil {
			return deployment.Reply{}, e
		}
		var n int
		e = db.QueryRow("SELECT count(*) FROM pending_deployment WHERE job_id=?", w.transport.cfg.Job).Scan(&n)
		db.Close()
		if e != nil || n != 1 {
			return deployment.Reply{}, errors.New("PENDING_WITNESS")
		}
		if _, e = w.transport.call("pending", nil); e != nil {
			return deployment.Reply{}, e
		}
		w.observed = true
	}
	return w.socket.Exchange(ctx, req)
}
func waitSocket(path string) error {
	for i := 0; i < 300; i++ {
		if st, e := os.Stat(path); e == nil && st.Mode()&os.ModeSocket != 0 {
			return nil
		}
		time.Sleep(time.Millisecond * 10)
	}
	return errors.New("SOCKET_START")
}
func run() error {
	if os.Geteuid() == 0 || len(os.Args) != 2 || os.Args[1] != "--synthetic-pipe" {
		return errors.New("TEST_ONLY")
	}
	scan := bufio.NewScanner(os.Stdin)
	scan.Buffer(make([]byte, 4096), 2<<20)
	var s setup
	if !scan.Scan() || json.Unmarshal(scan.Bytes(), &s) != nil {
		return errors.New("SETUP")
	}
	profile := sono.Profile()
	if profile.Product != s.Product {
		profile = secondary.Profile()
	}
	if profile.Product != s.Product {
		return errors.New("PROFILE")
	}
	profile.Artifact = s.Artifact
	dir, e := os.MkdirTemp("", "zop3-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(dir)
	peer := unixpeer.Peer{UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
	endpoint := func(n string) unixpeer.Endpoint { return unixpeer.Endpoint{Path: filepath.Join(dir, n), Peer: peer} }
	auth := rand.Text() + rand.Text()
	app, e := simulator.Open(filepath.Join(dir, "product.db"), profile, auth, time.Now)
	if e != nil {
		return e
	}
	defer app.Close()
	cfg := policy.Config{Scope: signing.Scope{Tenant: s.Identity.Tenant, Site: s.Identity.Site}, Installation: s.Identity.Installation, Agent: s.Identity.Agent, Version: "1.2.0", Trust: signing.Trust{}, TestArtifacts: map[string]string{s.Artifact: s.Product}, BackupForJob: map[string]policy.BackupRequirement{s.Job: {Product: s.Product, Artifact: s.Artifact, BackupManifest: s.BackupDigest, Signer: s.BackupKid, Categories: []string{s.Category}}}}
	for _, k := range s.Trust {
		pub, err := x509.ParsePKIXPublicKey(k.Public)
		if err != nil {
			return err
		}
		cfg.Trust[k.Kid] = signing.Key{Public: pub, Purpose: k.Purpose, Alg: k.Alg, Scope: signing.Scope{Tenant: k.Tenant, Site: k.Site}}
	}
	client := pl.Client{Endpoint: endpoint("app.sock"), HealthyCode: profile.Healthy, KnownLocks: []string{profile.Lock}}
	writer := client
	writer.WriteToken = auth
	h, e := helper.OpenSimulation(filepath.Join(dir, "helper.db"), cfg, peer.UID, peer.GID, time.Now, pl.Registry{s.Product: writer})
	if e != nil {
		return e
	}
	defer h.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	wg.Go(func() { _ = app.Serve(ctx, client.Endpoint.Path, peer) })
	wg.Go(func() { _ = h.Serve(ctx, endpoint("h.sock").Path) })
	for _, n := range []string{"app.sock", "h.sock"} {
		if e = waitSocket(endpoint(n).Path); e != nil {
			return e
		}
	}
	priv, e := x509.ParsePKCS8PrivateKey(s.ReceiptPrivate)
	if e != nil {
		return e
	}
	progress, e := x509.ParsePKCS8PrivateKey(s.ProgressPrivate)
	if e != nil {
		return e
	}
	progressLedger, e := store.Open(filepath.Join(dir, "progress.db"))
	if e != nil {
		return e
	}
	defer progressLedger.Close()
	if _, e = progressLedger.Exec("CREATE TABLE IF NOT EXISTS event(binding TEXT PRIMARY KEY,wire BLOB NOT NULL)"); e != nil {
		return e
	}
	pipe := &pipe{progressLedger: progressLedger, in: scan, out: json.NewEncoder(os.Stdout), cfg: s, policy: cfg, progress: progress}
	witness := &pendingWitness{socket: deployment.Socket{Endpoint: endpoint("h.sock")}, transport: pipe, ledger: filepath.Join(dir, "agent.db")}
	execute := func() error {
		a, err := agent.OpenConnected(filepath.Join(dir, "agent.db"), s.Identity, cfg, keys{s.ReceiptKid, priv}, nil, pl.Registry{s.Product: client}, witness)
		if err != nil {
			return err
		}
		defer a.Close()
		return a.Run(ctx, pipe, func() agent.Sample {
			now := time.Now()
			return agent.Sample{UTC: now.UTC(), Boot: "synthetic-boot", Monotonic: now.UnixNano(), Trusted: true}
		})
	}
	e = execute()
	if e != nil && e.Error() == "RECEIPT_OFFLINE" {
		if e = app.SetActivity(true, true, ""); e != nil {
			return e
		}
		if ok, err := client.ReadyVersion(ctx, profile.Release); err != nil || !ok {
			return errors.New("OFFLINE_CONTINUITY")
		}
		e = execute()
	}
	if !errors.Is(e, io.EOF) {
		return fmt.Errorf("agent: %w", e)
	}
	counts := map[string]int{}
	for _, q := range []struct{ name, file, query string }{{"effects", "product.db", "SELECT count(*) FROM effect"}, {"helperGrants", "helper.db", "SELECT count(*) FROM consumed_grant"}, {"agentGrants", "agent.db", "SELECT count(*) FROM approval_consumption"}, {"acknowledged", "agent.db", "SELECT count(*) FROM execution_receipt WHERE acknowledged_at IS NOT NULL"}} {
		db, err := store.Open(filepath.Join(dir, q.file))
		if err != nil {
			return err
		}
		var n int
		err = db.QueryRow(q.query).Scan(&n)
		db.Close()
		if err != nil {
			return err
		}
		counts[q.name] = n
	}
	return pipe.out.Encode(map[string]any{"Op": "done", "Counts": counts})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
