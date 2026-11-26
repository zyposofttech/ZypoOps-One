//go:build linux

package sprint2

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-policy"
	"zyposoft.local/operations/packages/local-store"
	pl "zyposoft.local/operations/packages/product-local"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/packages/unixpeer"
	secondary "zyposoft.local/operations/products/mock-secondary"
	"zyposoft.local/operations/products/simulator"
	sono "zyposoft.local/operations/products/zyposono-sim"
	"zyposoft.local/operations/services/local-helper"
	"zyposoft.local/operations/services/site-agent"
	"zyposoft.local/operations/tests/fixtures"
)

var helperToken = rand.Text() + rand.Text()
var peer = unixpeer.Peer{UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}

func endpoint(path string) unixpeer.Endpoint { return unixpeer.Endpoint{Path: path, Peer: peer} }

type plan struct {
	site     *fixture.Site
	profiles []simulator.Profile
	jobs     []policy.Bundle
	apps     pl.Registry
}

func makePlan(dir string, writable bool) plan {
	return productPlan(dir, writable, []simulator.Profile{sono.Profile(), secondary.Profile()})
}
func productPlan(dir string, writable bool, profiles []simulator.Profile) plan {
	site := fixture.NewSite()
	p := plan{site: site, profiles: profiles, apps: pl.Registry{}}
	for i, profile := range p.profiles {
		category := "APP_CONFIG"
		if profile.Product == secondary.Profile().Product {
			category = "MANAGEMENT_AGENT"
		}
		job := site.Deployment(i+1, profile.Product, profile.Lock, profile.Release, category, profile.Product)
		var err error
		profile.Artifact, err = policy.Digest(job.Manifest)
		if err != nil {
			panic(err)
		}
		p.profiles[i] = profile
		p.jobs = append(p.jobs, job)
		client := pl.Client{Endpoint: endpoint(filepath.Join(dir, profile.Product+".sock")), HealthyCode: profile.Healthy, KnownLocks: []string{profile.Lock}}
		if writable {
			client.WriteToken = helperToken
		}
		p.apps[profile.Product] = client
	}
	return p
}
func ident() agent.Identity {
	return agent.Identity{Tenant: fixture.Scope.Tenant, Site: fixture.Scope.Site, Installation: fixture.Install, Agent: fixture.AgentID, HostHash: strings.Repeat("a", 64)}
}
func clock() agent.Sample {
	return agent.Sample{UTC: fixture.Now, Boot: "simulated-boot", Monotonic: 1, Trusted: true}
}
func startApps(t *testing.T, dir string, p plan) []*simulator.App {
	t.Helper()
	return startAppsAt(t, dir, p, func() time.Time { return fixture.Now })
}
func startAppsAt(t *testing.T, dir string, p plan, now func() time.Time) []*simulator.App {
	t.Helper()
	var apps []*simulator.App
	for _, profile := range p.profiles {
		a, err := simulator.Open(filepath.Join(dir, profile.Product+".db"), profile, helperToken, now)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, profile.Product+".sock")
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- a.Serve(ctx, path, peer) }()
		t.Cleanup(func() {
			cancel()
			if err := <-done; err != nil {
				t.Error(err)
			}
			a.Close()
		})
		waitSocket(t, path)
		apps = append(apps, a)
	}
	return apps
}
func waitSocket(t *testing.T, path string) {
	t.Helper()
	for i := 0; i < 250; i++ {
		if c, err := net.DialTimeout("unix", path, 100*time.Millisecond); err == nil {
			c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("socket unavailable", path)
}
func testDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "zop2-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}
func serveHelper(t *testing.T, dir string, p plan) deployment.Socket {
	t.Helper()
	h, err := helper.OpenSimulation(filepath.Join(dir, "helper.db"), p.site.Config, peer.UID, peer.GID, func() time.Time { return fixture.Now }, p.apps)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	path := filepath.Join(dir, "h.sock")
	go func() { done <- h.Serve(ctx, path) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
		h.Close()
	})
	waitSocket(t, path)
	return deployment.Socket{Endpoint: endpoint(path)}
}
func scalar(t *testing.T, path, query string) int {
	t.Helper()
	db, err := localstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err = db.QueryRow(query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func helperReply(t *testing.T, s deployment.Socket, operation string, b policy.Bundle) deployment.Reply {
	t.Helper()
	r, err := s.Exchange(context.Background(), deployment.Request{Operation: operation, Bundle: b})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Re-exec the same test binary. Each child reopens only its role's database.
func TestServiceProcess(t *testing.T) {
	role := os.Getenv("ZOP2_ROLE")
	if role == "" {
		t.Skip("child-process entry point")
	}
	dir := os.Getenv("ZOP2_DIR")
	if role == "helper" || role == "product" {
		helperToken = os.Getenv("ZOP2_PRODUCT_TOKEN")
	}
	p := makePlan(dir, role == "helper")
	if kid := os.Getenv("ZOP2_REVOKE"); kid != "" {
		key := p.site.Config.Trust[kid]
		key.Revoked = true
		p.site.Config.Trust[kid] = key
	}
	if os.Getenv("ZOP2_ROTATE") == "1" {
		delete(p.site.Config.Trust, "mock-job")
	}
	now := func() time.Time {
		if raw := os.Getenv("ZOP2_TIME"); raw != "" {
			v, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
		return fixture.Now
	}
	cut := processBarrier(t)
	switch role {
	case "product":
		profile := p.profiles[0]
		app, err := simulator.Open(filepath.Join(dir, profile.Product+".db"), profile, helperToken, now)
		if err != nil {
			t.Fatal(err)
		}
		defer app.Close()
		// A test-only local event source remains available without either management process.
		events := make(chan os.Signal, 1)
		signal.Notify(events, syscall.SIGUSR1)
		defer signal.Stop(events)
		go func() {
			for range events {
				if err := app.SetActivity(true, true, ""); err != nil {
					t.Error(err)
					return
				}
				cut("activity")
			}
		}()
		if err = app.Serve(context.Background(), filepath.Join(dir, profile.Product+".sock"), peer); err != nil {
			t.Fatal(err)
		}
	case "helper":
		for key, app := range p.apps {
			p.apps[key] = witnessProduct{app, cut}
		}
		h, err := helper.OpenSimulation(filepath.Join(dir, "helper.db"), p.site.Config, peer.UID, peer.GID, now, p.apps)
		if err != nil {
			t.Fatal(err)
		}
		defer h.Close()
		if err = h.Serve(context.Background(), filepath.Join(dir, "h.sock")); err != nil {
			t.Fatal(err)
		}
	case "agent":
		connection := witnessSocket{deployment.Socket{Endpoint: endpoint(os.Getenv("ZOP2_HELPER"))}, cut}
		a, err := agent.OpenConnected(filepath.Join(dir, "agent.db"), ident(), p.site.Config, p.site, nil, p.apps, connection)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		err = a.Run(context.Background(), Outbound{os.Getenv("ZOP2_CENTRAL")}, func() agent.Sample { s := clock(); s.UTC = now(); return s })
		if os.Getenv("ZOP2_EXPECT_ACK_LOSS") == "1" {
			if err == nil || err.Error() != "RECEIPT_OFFLINE" {
				t.Fatal("expected lost acknowledgement", err)
			}
			return
		}
		if expected := os.Getenv("ZOP2_EXPECT_ERROR"); expected != "" {
			if err == nil || err.Error() != expected {
				t.Fatalf("expected %s, got %v", expected, err)
			}
			return
		}
		if !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
	default:
		t.Fatal("invalid test role")
	}
}

type child struct {
	cmd    *exec.Cmd
	done   chan error
	log    bytes.Buffer
	cancel context.CancelFunc
	waited bool
}

func launch(t *testing.T, dir, role, helperPath, central string, loseAck bool, options ...string) *child {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	c := &child{cmd: exec.CommandContext(ctx, os.Args[0], "-test.run=^TestServiceProcess$"), done: make(chan error, 1), cancel: cancel}
	c.cmd.Env = append(os.Environ(), "ZOP2_DIR="+dir, "ZOP2_ROLE="+role, "ZOP2_HELPER="+helperPath, "ZOP2_CENTRAL="+central)
	c.cmd.Env = append(c.cmd.Env, options...)
	if role == "helper" || role == "product" {
		c.cmd.Env = append(c.cmd.Env, "ZOP2_PRODUCT_TOKEN="+helperToken)
	}
	if loseAck {
		c.cmd.Env = append(c.cmd.Env, "ZOP2_EXPECT_ACK_LOSS=1")
	}
	c.cmd.Stdout = &c.log
	c.cmd.Stderr = &c.log
	if err := c.cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	go func() { c.done <- c.cmd.Wait() }()
	t.Cleanup(func() {
		if !c.waited {
			c.kill(t)
		}
		cancel()
	})
	return c
}
func (c *child) kill(t *testing.T) {
	t.Helper()
	if err := c.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err := <-c.done
	c.waited = true
	c.cancel()
	if err == nil {
		t.Fatal("child survived kill")
	}
}
func (c *child) wait(t *testing.T) {
	t.Helper()
	err := <-c.done
	c.waited = true
	c.cancel()
	if err != nil {
		t.Fatalf("child: %v %s", err, c.log.String())
	}
}
func TestCommittedDisconnect(t *testing.T) {
	dir := testDir(t)
	p := makePlan(dir, false)
	startApps(t, dir, p)
	ctx := context.Background()
	central, err := OpenCentral(filepath.Join(dir, "central.db"), fixture.Scope, p.site.Config.Trust, primaryEnrolment())
	if err != nil {
		t.Fatal(err)
	}
	defer central.Close()
	for _, b := range p.jobs {
		if err = central.Submit(b); err != nil {
			t.Fatal(err)
		}
	}
	var requests atomic.Int32
	var loseAck atomic.Bool
	loseAck.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/receipt" {
			requests.Add(1)
			if loseAck.CompareAndSwap(true, false) {
				raw, e := io.ReadAll(r.Body)
				if e != nil || central.accept(raw) != nil {
					w.WriteHeader(409)
					return
				}
				conn, _, e := w.(http.Hijacker).Hijack()
				if e == nil {
					conn.Close()
				}
				return
			}
		}
		central.ServeHTTP(w, r)
	}))
	defer server.Close()
	hpath := filepath.Join(dir, "h.sock")
	h := launch(t, dir, "helper", "", "", false)
	waitSocket(t, hpath)
	// Forward real framed socket traffic. Hold only the first successful execute response after the helper committed it.
	proxy := filepath.Join(dir, "relay.sock")
	l, err := unixpeer.Listen(proxy)
	if err != nil {
		t.Fatal(err)
	}
	committed := make(chan deployment.Result, 1)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	defer l.Close()
	go func() {
		for {
			conn, e := l.AcceptUnix()
			if e != nil {
				return
			}
			func() {
				defer conn.Close()
				var n uint32
				if binary.Read(conn, binary.BigEndian, &n) != nil || n > 1<<20 {
					return
				}
				raw := make([]byte, n)
				if _, e = io.ReadFull(conn, raw); e != nil {
					return
				}
				var req deployment.Request
				if json.Unmarshal(raw, &req) != nil {
					return
				}
				reply, e := (deployment.Socket{Endpoint: endpoint(hpath)}).Exchange(ctx, req)
				if e != nil {
					return
				}
				if req.Operation == "execute" && reply.Status == "DONE" {
					committed <- *reply.Result
					<-release
					return
				}
				_ = json.NewEncoder(conn).Encode(reply)
			}()
		}
	}()
	a := launch(t, dir, "agent", proxy, server.URL, false)
	var original deployment.Result
	select {
	case original = <-committed:
	case <-time.After(8 * time.Second):
		t.Fatal("no committed helper response")
	}
	a.kill(t)
	h.kill(t)
	unblock()
	if n := scalar(t, filepath.Join(dir, "agent.db"), "SELECT count(*) FROM execution_receipt"); n != 0 {
		t.Fatal("agent had confirmation", n)
	}
	for i, profile := range p.profiles {
		want := profile.Initial
		if i == 0 {
			want = profile.Release
		}
		ok, e := p.apps[profile.Product].ReadyVersion(ctx, want)
		if e != nil || !ok {
			t.Fatal("product unavailable during outage", profile.Product, e)
		}
	}
	h = launch(t, dir, "helper", "", "", false)
	waitSocket(t, hpath)
	launch(t, dir, "agent", hpath, server.URL, true).wait(t)
	launch(t, dir, "agent", hpath, server.URL, false).wait(t)
	if requests.Load() != 3 {
		t.Fatal("delivery count", requests.Load())
	}
	for _, profile := range p.profiles {
		if n := scalar(t, filepath.Join(dir, profile.Product+".db"), "SELECT count(*) FROM effect"); n != 1 {
			t.Fatal(profile.Product, n)
		}
		ok, e := p.apps[profile.Product].ReadyVersion(ctx, profile.Release)
		if e != nil || !ok {
			t.Fatal(profile.Product, e)
		}
	}
	for _, v := range []struct {
		db, sql string
		want    int
	}{{"helper.db", "SELECT count(*) FROM deployment_execution WHERE result IS NOT NULL", 2}, {"helper.db", "SELECT count(*) FROM consumed_grant", 2}, {"agent.db", "SELECT count(*) FROM approval_consumption", 2}, {"agent.db", "SELECT count(*) FROM execution_receipt WHERE acknowledged_at IS NOT NULL", 2}, {"central.db", "SELECT count(*) FROM terminal", 2}} {
		if n := scalar(t, filepath.Join(dir, v.db), v.sql); n != v.want {
			t.Fatal(v, n)
		}
	}
	var receipt []byte
	if err = central.db.QueryRow("SELECT receipt FROM terminal WHERE job=?", original.Job).Scan(&receipt); err != nil {
		t.Fatal(err)
	}
	doc, err := signing.Verify("MC-04", receipt, fixture.Scope, p.site.Config.Trust)
	if err != nil {
		t.Fatal(err)
	}
	body := doc["body"].(signing.Object)
	if body["execution_id"] != original.Execution || body["observed_sha256"] != original.Artifact || body["state"] != "SUCCEEDED" {
		t.Fatal(body)
	}
	h.kill(t)
}
