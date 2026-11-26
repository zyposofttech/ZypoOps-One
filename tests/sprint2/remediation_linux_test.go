//go:build linux

package sprint2

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-policy"
	"zyposoft.local/operations/packages/local-store"
	pl "zyposoft.local/operations/packages/product-local"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/packages/unixpeer"
	"zyposoft.local/operations/services/local-helper"
	"zyposoft.local/operations/services/site-agent"
	"zyposoft.local/operations/tests/fixtures"
)

type gate struct {
	entered, release chan struct{}
	once             sync.Once
}

func newGate() *gate { return &gate{entered: make(chan struct{}), release: make(chan struct{})} }
func (g *gate) pause(ctx context.Context) error {
	g.once.Do(func() { close(g.entered) })
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-g.release:
		return nil
	}
}
func (g *gate) wait(t *testing.T) {
	t.Helper()
	select {
	case <-g.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("barrier not reached")
	}
}

type pausedProduct struct {
	pl.Application
	check, apply *gate
}

func (p pausedProduct) Preflight(ctx context.Context, locks []string) error {
	if err := p.Application.Preflight(ctx, locks); err != nil {
		return err
	}
	if p.check != nil {
		return p.check.pause(ctx)
	}
	return nil
}
func (p pausedProduct) Apply(ctx context.Context, a pl.Activation) (pl.Observation, error) {
	if p.apply != nil {
		if err := p.apply.pause(ctx); err != nil {
			return pl.Observation{}, err
		}
	}
	return p.Application.Apply(ctx, a)
}
func limitedHelper(t *testing.T, dir string, p plan, limit int, now func() time.Time) (deployment.Socket, func()) {
	t.Helper()
	h, err := helper.OpenSimulation(filepath.Join(dir, "helper.db"), p.site.Config, peer.UID, peer.GID, now, p.apps)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	end := make(chan error, 1)
	path := filepath.Join(dir, "h.sock")
	go func() { end <- h.ServeWithLimit(ctx, path, limit) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, e := os.Stat(path); e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("listener unavailable")
		}
		time.Sleep(time.Millisecond)
	}
	stop := sync.OnceFunc(func() {
		cancel()
		if err := <-end; err != nil {
			t.Error(err)
		}
		h.Close()
	})
	t.Cleanup(stop)
	return deployment.Socket{Endpoint: endpoint(path)}, stop
}

type exchange struct {
	reply deployment.Reply
	err   error
}

func submitAsync(ctx context.Context, s deployment.Socket, op string, b policy.Bundle) <-chan exchange {
	done := make(chan exchange, 1)
	go func() { r, e := s.Exchange(ctx, deployment.Request{Operation: op, Bundle: b}); done <- exchange{r, e} }()
	return done
}
func expectCount(t *testing.T, dir, file, table string, want int) {
	t.Helper()
	if got := scalar(t, filepath.Join(dir, file), "SELECT count(*) FROM "+table); got != want {
		t.Fatalf("%s/%s=%d, want %d", file, table, got, want)
	}
}

func TestOverlappingAdmissions(t *testing.T) {
	dir := testDir(t)
	p := makePlan(dir, true)
	startApps(t, dir, p)
	one, two, mutation := newGate(), newGate(), newGate()
	first, second := p.profiles[0].Product, p.profiles[1].Product
	p.apps[first] = pausedProduct{p.apps[first], one, mutation}
	p.apps[second] = pausedProduct{p.apps[second], two, nil}
	s, stop := limitedHelper(t, dir, p, 8, func() time.Time { return fixture.Now })
	ctx := context.Background()
	a := submitAsync(ctx, s, "execute", p.jobs[0])
	one.wait(t)
	b := submitAsync(ctx, s, "execute", p.jobs[1])
	two.wait(t) // Two distinct live handlers reached product preflight simultaneously.
	close(one.release)
	mutation.wait(t)
	close(two.release)
	if r := <-b; r.err != nil || r.reply.Status != deployment.Retry {
		t.Fatal("competing product", r)
	}
	lookup := submitAsync(ctx, s, "lookup", p.jobs[0])
	abort := submitAsync(ctx, s, "abort", p.jobs[0])
	duplicate := submitAsync(ctx, s, "execute", p.jobs[0])
	close(mutation.release)
	for _, done := range []<-chan exchange{a, lookup, abort, duplicate} {
		r := <-done
		if r.err != nil || r.reply.Result == nil || r.reply.Result.State != "SUCCEEDED" {
			t.Fatal(r)
		}
	}
	stop()
	s, _ = limitedHelper(t, dir, p, 8, func() time.Time { return fixture.Now })
	r := helperReply(t, s, "lookup", p.jobs[0])
	if r.Result == nil || r.Result.State != "SUCCEEDED" {
		t.Fatal(r)
	}
	expectCount(t, dir, first+".db", "effect", 1)
	expectCount(t, dir, second+".db", "effect", 0)
	expectCount(t, dir, "helper.db", "consumed_grant", 1)
	expectCount(t, dir, "helper.db", "deployment_execution", 1)
}

func TestBoundedStalledConnections(t *testing.T) {
	dir := testDir(t)
	p := makePlan(dir, true)
	startApps(t, dir, p)
	key := p.profiles[0].Product
	hold := newGate()
	p.apps[key] = pausedProduct{p.apps[key], hold, nil}
	s, stop := limitedHelper(t, dir, p, 2, func() time.Time { return fixture.Now })
	slow, err := net.Dial("unix", s.Endpoint.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer slow.Close()
	if err = binary.Write(slow, binary.BigEndian, uint16(0)); err != nil {
		t.Fatal(err)
	} // half a frame header
	if r := helperReply(t, s, "lookup", p.jobs[0]); r.Status != deployment.Absent {
		t.Fatal(r)
	}
	active := submitAsync(context.Background(), s, "execute", p.jobs[0])
	hold.wait(t)
	if _, err = s.Exchange(context.Background(), deployment.Request{Operation: "lookup", Bundle: p.jobs[1]}); err == nil {
		t.Fatal("accepted third worker")
	}
	stop()
	if r := <-active; r.err == nil && r.reply.Status == deployment.Done {
		t.Fatal(r)
	}
	expectCount(t, dir, "helper.db", "consumed_grant", 0)
	expectCount(t, dir, key+".db", "effect", 0)
}

// A real LOCAL-01 HTTP server withholds the locks response. An independent SQLite
// connection must obtain a writer while the preflight is outside both ledgers.
func TestPreflightTransactions(t *testing.T) {
	for _, role := range []string{"agent", "helper"} {
		for _, fault := range []string{"insert failure", "cancel", "expiry", "clock rollback"} {
			t.Run(role+"/"+fault, func(t *testing.T) {
				dir := testDir(t)
				p := makePlan(dir, true)
				apps := startApps(t, dir, p)
				key := p.profiles[0].Product
				hold := newGate()
				l, err := unixpeer.Listen(filepath.Join(dir, "slow.sock"))
				if err != nil {
					t.Fatal(err)
				}
				server := http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/local/v1/locks" {
						if hold.pause(r.Context()) != nil {
							return
						}
					}
					apps[0].ServeHTTP(w, r)
				})}
				go server.Serve(unixpeer.Listener{OwnedListener: l, Allowed: peer})
				defer server.Close()
				client := p.apps[key].(pl.Client)
				client.Endpoint = endpoint(filepath.Join(dir, "slow.sock"))
				p.apps[key] = client
				b := p.jobs[0]
				if fault == "expiry" {
					g := fixture.Doc(b.Grants[0])
					g["body"].(signing.Object)["valid_until"] = fixture.Now.Add(200 * time.Millisecond).Format(time.RFC3339Nano)
					b.Grants[0] = p.site.SignGrant(g)
				}
				var advanced atomic.Bool
				now := func() time.Time {
					if advanced.Load() {
						if fault == "clock rollback" {
							return fixture.Now.Add(-time.Second)
						}
						return fixture.Now.Add(time.Second)
					}
					return fixture.Now
				}
				socket, stop := limitedHelper(t, dir, p, 8, now)
				path := filepath.Join(dir, role+".db")
				table := "deployment_execution"
				ledger := "consumed_grant"
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				finished := make(chan error, 1)
				if role == "agent" {
					table = "pending_deployment"
					ledger = "approval_consumption"
					a, e := agent.OpenConnected(path, ident(), p.site.Config, p.site, nil, p.apps, socket)
					if e != nil {
						t.Fatal(e)
					}
					defer a.Close()
					go func() {
						finished <- a.Run(ctx, singleJob{b}, func() agent.Sample { s := clock(); s.UTC = now(); return s })
					}()
				} else {
					go func() {
						r, e := socket.Exchange(ctx, deployment.Request{Operation: "execute", Bundle: b})
						if e == nil && r.Status != deployment.Done {
							e = errors.New(string(r.Status))
						}
						finished <- e
					}()
				}
				hold.wait(t)
				db, e := localstore.Open(path)
				if e != nil {
					t.Fatal(e)
				}
				defer db.Close()
				if _, e = db.Exec("PRAGMA busy_timeout=80"); e != nil {
					t.Fatal(e)
				}
				if _, e = db.Exec("CREATE TABLE probe(value INTEGER)"); e != nil {
					t.Fatal("writer blocked during preflight", e)
				}
				switch fault {
				case "insert failure":
					if _, e = db.Exec("CREATE TRIGGER fail_insert BEFORE INSERT ON " + table + " BEGIN SELECT RAISE(ABORT,'test persistence failure'); END"); e != nil {
						t.Fatal(e)
					}
				case "cancel":
					if role == "helper" {
						stop()
					} else {
						cancel()
					}
				case "expiry", "clock rollback":
					advanced.Store(true)
				}
				close(hold.release)
				if e := <-finished; e == nil {
					t.Fatal("admission succeeded", fault)
				}
				if fault == "clock rollback" {
					query := "SELECT count(*) FROM clock_watermark WHERE clock_confidence='UNTRUSTED'"
					if role == "helper" {
						query = "SELECT count(*) FROM time_floor WHERE blocked=1"
					}
					if scalar(t, path, query) != 1 {
						t.Fatal("rollback latch lost")
					}
				}
				expectCount(t, dir, role+".db", ledger, 0)
				expectCount(t, dir, key+".db", "effect", 0)
			})
		}
	}
}

type singleJob struct{ b policy.Bundle }

func (j singleJob) Next(context.Context) (policy.Bundle, error) { return j.b, nil }
func (j singleJob) Receipt(context.Context, []byte) error       { return errors.New("unexpected receipt") }

func TestUnknownStageMigration(t *testing.T) {
	dir := testDir(t)
	p := makePlan(dir, true)
	apps := startApps(t, dir, p)
	if err := apps[0].SetActivity(false, true, "POST_VERIFY_OFFLINE"); err != nil {
		t.Fatal(err)
	}
	socket := serveHelper(t, dir, p)
	path := filepath.Join(dir, "agent.db")
	open := func() *agent.Agent {
		a, e := agent.OpenConnected(path, ident(), p.site.Config, p.site, nil, p.apps, socket)
		if e != nil {
			t.Fatal(e)
		}
		return a
	}
	a := open()
	wire, e := a.Handle(context.Background(), p.jobs[0], clock())
	if e != nil {
		t.Fatal(e)
	}
	a.Close()
	c, e := OpenCentral(filepath.Join(dir, "central.db"), fixture.Scope, p.site.Config.Trust, primaryEnrolment())
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	if e = c.Submit(p.jobs[0]); e != nil {
		t.Fatal(e)
	}
	if e = c.accept(wire); e != nil {
		t.Fatal(e)
	}
	r := helperReply(t, socket, "lookup", p.jobs[0])
	if r.Result == nil || r.Result.State != "UNKNOWN" {
		t.Fatal(r)
	}
	db, e := localstore.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	for pass := 0; pass < 2; pass++ {
		var stage, job, out, raw string
		if e = db.QueryRow("SELECT s.state,j.state,r.terminal_state,r.signed_receipt FROM job_stage s JOIN received_job j USING(job_id) JOIN execution_receipt r USING(job_id)").Scan(&stage, &job, &out, &raw); e != nil || stage != "UNKNOWN" || job != "UNKNOWN" || out != "UNKNOWN" || raw != string(wire) {
			t.Fatal(stage, job, out, e)
		}
		if pass == 0 {
			if _, e = db.Exec("UPDATE job_stage SET state='FAILED'"); e != nil {
				t.Fatal(e)
			}
			open().Close()
		}
	}
	var centralRaw []byte
	if e = c.db.QueryRow("SELECT receipt FROM terminal").Scan(&centralRaw); e != nil || string(centralRaw) != string(wire) {
		t.Fatal(e)
	}
}
