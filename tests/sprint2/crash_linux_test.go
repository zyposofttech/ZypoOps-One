//go:build linux

package sprint2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-store"
	pl "zyposoft.local/operations/packages/product-local"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/tests/fixtures"
)

// Every barrier brackets a real operation. The parent inspects committed stores
// through fresh connections while the child is blocked on the witness HTTP call.
type barrier func(string)

func processBarrier(t *testing.T) barrier {
	return func(point string) {
		if point != os.Getenv("ZOP2_CUT") {
			return
		}
		res, err := http.Get(os.Getenv("ZOP2_WITNESS") + "/" + point)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
	}
}

type witnessProduct struct {
	pl.Application
	cut barrier
}

func (p witnessProduct) Lookup(ctx context.Context, a pl.Activation) (pl.Observation, error) {
	p.cut("reserved")
	return p.Application.Lookup(ctx, a)
}
func (p witnessProduct) Apply(ctx context.Context, a pl.Activation) (pl.Observation, error) {
	o, err := p.Application.Apply(ctx, a)
	if err == nil && o.Phase == "APPLIED" {
		p.cut("applied")
	}
	return o, err
}

type witnessSocket struct {
	deployment.Socket
	cut barrier
}

func (s witnessSocket) Exchange(ctx context.Context, q deployment.Request) (deployment.Reply, error) {
	if q.Operation == "lookup" {
		s.cut("pending")
	}
	r, err := s.Socket.Exchange(ctx, q)
	if err == nil && r.Status == "DONE" {
		s.cut("reply")
	}
	return r, err
}

func TestCrashMatrix(t *testing.T) {
	cases := []struct {
		id, cut                 string
		advance                 time.Duration
		wantEffects             int
		productKill, agentFirst bool
		disconnects             int
	}{
		{"CR01", "pending", 0, 1, false, false, 0},
		{"CR02", "reserved", 0, 1, false, false, 0},
		{"CR03", "applied", 0, 1, false, false, 0},
		{"CR05", "presend", 0, 1, false, false, 0},
		{"CR06", "preack", 0, 1, false, false, 0},
		{"CR07", "applied", 0, 1, true, false, 0},
		{"CR08", "reserved", 0, 1, false, true, 0},
		{"CR09-expired-committed", "applied", 2 * 365 * 24 * time.Hour, 1, false, false, 0},
		{"CR09-expired-reservation", "reserved", 2 * 365 * 24 * time.Hour, 0, false, false, 0},
		{"CR09-stale-backup", "reserved", 25 * time.Hour, 0, false, false, 0},
		{"CR09-revoked-grant", "reserved", 0, 0, false, false, 0},
		{"CR09-revoked-ops-job-committed", "applied", 0, 1, false, false, 0},
		{"CR09-revoked-ops-job-reserved", "reserved", 0, 0, false, false, 0},
		{"CR09-revoked-ops-job-before-helper-admission", "pending", 0, 0, false, false, 0},
		{"CR09-revoked-ops-job-terminal", "reply", 0, 1, false, false, 0},
		{"CR09-rotated-ops-job-committed", "applied", 0, 1, false, false, 0},
		{"CR09-revoked-ops-job-repeated-recovery", "applied", 0, 1, false, false, 2},
		{"CR09-revoked-ops-job-outbox", "presend", 0, 1, false, false, 0},
		{"CR09-revoked-receipt-key", "applied", 0, 1, false, false, 0},
		{"CR09-revoked-ops-job-observation-offline", "applied", 0, 1, false, false, 0},
		{"CR10", "applied", 0, 1, false, false, 2},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			ctx := context.Background()
			dir := testDir(t)
			p := makePlan(dir, true)
			profile := p.profiles[0]
			b := p.jobs[0]
			binding, err := deployment.BundleDigest(b)
			if err != nil {
				t.Fatal(err)
			}
			job := fixture.Doc(b.Job)["body"].(signing.Object)["job_id"].(string)
			execution := deployment.ExecutionID(binding)
			central, err := OpenCentral(filepath.Join(dir, "central.db"), fixture.Scope, p.site.Config.Trust, primaryEnrolment())
			if err != nil {
				t.Fatal(err)
			}
			defer central.Close()
			if err = central.Submit(b); err != nil {
				t.Fatal(err)
			}
			hit := make(chan string, 1)
			release := make(chan struct{})
			pause := func(r *http.Request, point string) {
				select {
				case hit <- point:
				case <-r.Context().Done():
					return
				}
				select {
				case <-r.Context().Done():
				case <-release:
				}
			}
			witness := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { pause(r, r.URL.Path[1:]) }))
			defer witness.Close()
			var intercepted atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/receipt" && (tc.cut == "preack" || tc.cut == "presend") && intercepted.CompareAndSwap(false, true) {
					if tc.cut == "preack" {
						central.ServeHTTP(w, r)
					}
					pause(r, tc.cut)
					return
				}
				central.ServeHTTP(w, r)
			}))
			defer server.Close()
			defer close(release)
			wait := func(point string) {
				t.Helper()
				select {
				case got := <-hit:
					if got != point {
						t.Fatal("wrong barrier", point, got)
					}
				case <-time.After(12 * time.Second):
					t.Fatal("barrier timeout", point)
				}
			}
			cut := func(point string) []string { return []string{"ZOP2_CUT=" + point, "ZOP2_WITNESS=" + witness.URL} }
			hpath := filepath.Join(dir, "h.sock")
			ppath := filepath.Join(dir, profile.Product+".sock")
			product := launch(t, dir, "product", "", "", false)
			waitSocket(t, ppath)
			h := launch(t, dir, "helper", "", "", false, cut(tc.cut)...)
			waitSocket(t, hpath)
			a := launch(t, dir, "agent", hpath, server.URL, false, cut(tc.cut)...)
			wait(tc.cut)
			count := func(file, sql string, want int) {
				t.Helper()
				if n := scalar(t, filepath.Join(dir, file), sql); n != want {
					t.Fatalf("%s %s = %d, want %d", file, sql, n, want)
				}
			}
			checkRow := func(file, query string, want string, args ...any) {
				t.Helper()
				db, e := localstore.Open(filepath.Join(dir, file))
				if e != nil {
					t.Fatal(e)
				}
				defer db.Close()
				var value string
				if e = db.QueryRow(query, args...).Scan(&value); e != nil || value != want {
					t.Fatalf("%s %s = %q, want %q: %v", file, query, value, want, e)
				}
			}
			reserved, applied, receipt, accepted := 1, 0, 0, 0
			switch tc.cut {
			case "pending":
				reserved = 0
			case "applied", "reply":
				applied = 1
			case "presend", "preack":
				applied = 1
				receipt = 1
			}
			if tc.cut == "preack" {
				accepted = 1
			}
			checkRow("agent.db", "SELECT binding FROM pending_deployment WHERE job_id=?", binding, job)
			count("agent.db", "SELECT count(*) FROM approval_consumption", 1)
			count("helper.db", "SELECT count(*) FROM consumed_grant", reserved)
			helperTerminal := receipt
			if tc.cut == "reply" {
				helperTerminal = 1
			}
			count("helper.db", "SELECT count(*) FROM deployment_execution WHERE result IS NOT NULL", helperTerminal)
			count(profile.Product+".db", "SELECT count(*) FROM effect", applied)
			count("agent.db", "SELECT count(*) FROM execution_receipt", receipt)
			count("central.db", "SELECT count(*) FROM terminal", accepted)
			if reserved == 1 {
				checkRow("helper.db", "SELECT execution FROM deployment_execution WHERE job=? AND bundle=?", execution, job, binding)
			}
			version := profile.Initial
			if applied == 1 {
				version = profile.Release
			}
			if tc.productKill {
				product.kill(t)
				product = launch(t, dir, "product", "", "", false, cut("activity")...)
				waitSocket(t, ppath)
				if ok, e := p.apps[profile.Product].ReadyVersion(ctx, version); e != nil || !ok {
					t.Fatal("lost release on product restart", e)
				}
			}
			var unacked string
			if receipt == 1 {
				db, e := localstore.Open(filepath.Join(dir, "agent.db"))
				if e != nil {
					t.Fatal(e)
				}
				e = db.QueryRow("SELECT signed_receipt FROM execution_receipt WHERE job_id=?", job).Scan(&unacked)
				db.Close()
				if e != nil {
					t.Fatal(e)
				}
			}
			a.kill(t)
			h.kill(t)
			if ok, e := p.apps[profile.Product].ReadyVersion(ctx, version); e != nil || !ok {
				t.Fatal("activity unavailable during outage", e)
			}
			if tc.productKill {
				if err := product.cmd.Process.Signal(syscall.SIGUSR1); err != nil {
					t.Fatal(err)
				}
				wait("activity")
				count(profile.Product+".db", "SELECT count(*) FROM application WHERE active=1 AND ready=1", 1)
			}
			sample := "ZOP2_TIME=" + fixture.Now.Add(tc.advance).Format(time.RFC3339)
			// For agent-first startup, stop the restarted agent at its outbound call;
			// the first real helper dispatch happens only after its socket is rebound.
			if tc.agentFirst {
				a = launch(t, dir, "agent", hpath, server.URL, false, append(cut("pending"), sample)...)
				wait("pending")
			}
			recovery := []string{sample}
			if tc.id == "CR09-revoked-grant" {
				recovery = append(recovery, "ZOP2_REVOKE=mock-local")
			}
			if strings.Contains(tc.id, "revoked-ops-job") {
				recovery = append(recovery, "ZOP2_REVOKE=mock-job")
			}
			if strings.Contains(tc.id, "rotated-ops-job") {
				recovery = append(recovery, "ZOP2_ROTATE=1")
			}
			if tc.id == "CR09-revoked-receipt-key" {
				recovery = append(recovery, "ZOP2_REVOKE=mock-receipt")
			}
			h = launch(t, dir, "helper", "", "", false, recovery...)
			waitSocket(t, hpath)
			if reserved == 1 && helperTerminal == 0 {
				next := p.jobs[1]
				next.Job = p.site.Sign(fixture.Doc(next.Job), "rotated-job")
				r := helperReply(t, deployment.Socket{Endpoint: endpoint(hpath)}, "execute", next)
				if r.Status != deployment.Retry {
					t.Fatal("pending execution lost its admission hold", r)
				}
				count("helper.db", "SELECT count(*) FROM consumed_grant", 1)
			}
			if tc.agentFirst {
				release <- struct{}{}
			}
			for range tc.disconnects {
				a = launch(t, dir, "agent", hpath, server.URL, false, append(cut("reply"), recovery...)...)
				wait("reply")
				a.kill(t)
			}
			if tc.id == "CR09-revoked-ops-job-observation-offline" {
				product.kill(t)
				launch(t, dir, "agent", hpath, server.URL, false, append(recovery, "ZOP2_EXPECT_ERROR=HISTORICAL_RECOVERY_REQUIRED")...).wait(t)
				count("agent.db", "SELECT count(*) FROM received_job WHERE state='RUNNING'", 1)
				count("agent.db", "SELECT count(*) FROM deployment_recovery_hold", 1)
				count("helper.db", "SELECT count(*) FROM deployment_execution WHERE result IS NULL", 1)
				count(profile.Product+".db", "SELECT count(*) FROM effect", 1)
				count("central.db", "SELECT count(*) FROM terminal", 0)
				next := p.jobs[1]
				next.Job = p.site.Sign(fixture.Doc(next.Job), "rotated-job")
				if r := helperReply(t, deployment.Socket{Endpoint: endpoint(hpath)}, "execute", next); r.Status != deployment.Retry {
					t.Fatal("offline observation released hold", r)
				}
				product = launch(t, dir, "product", "", "", false)
				waitSocket(t, ppath)
			}
			if tc.id == "CR09-revoked-ops-job-before-helper-admission" || tc.id == "CR09-revoked-receipt-key" {
				reason := "HISTORICAL_RECOVERY_REQUIRED"
				if tc.id == "CR09-revoked-receipt-key" {
					reason = "UNTRUSTED_KEY"
				}
				launch(t, dir, "agent", hpath, server.URL, false, append(recovery, "ZOP2_EXPECT_ERROR="+reason)...).wait(t)
				count("agent.db", "SELECT count(*) FROM received_job WHERE state='RUNNING'", 1)
				count("agent.db", "SELECT count(*) FROM deployment_recovery_hold", 1)
				count("agent.db", "SELECT count(*) FROM approval_consumption", 1)
				count("helper.db", "SELECT count(*) FROM consumed_grant", reserved)
				count("helper.db", "SELECT count(*) FROM deployment_execution", reserved)
				count("agent.db", "SELECT count(*) FROM execution_receipt", 0)
				count("central.db", "SELECT count(*) FROM terminal", 0)
				if reserved == 0 {
					r := helperReply(t, deployment.Socket{Endpoint: endpoint(hpath)}, "execute", b)
					if r.Status != deployment.Denied {
						t.Fatal(r)
					}
				}
				count(profile.Product+".db", "SELECT count(*) FROM effect", tc.wantEffects)
				h.kill(t)
				product.kill(t)
				return
			}
			if !tc.agentFirst {
				a = launch(t, dir, "agent", hpath, server.URL, false, recovery...)
			}
			a.wait(t)
			count("agent.db", "SELECT count(*) FROM deployment_recovery_hold", 0)
			count("agent.db", "SELECT count(*) FROM approval_consumption", 1)
			count("helper.db", "SELECT count(*) FROM consumed_grant", 1)
			count("helper.db", "SELECT count(*) FROM deployment_execution", 1)
			count("central.db", "SELECT count(*) FROM terminal", 1)
			count("agent.db", "SELECT count(*) FROM execution_receipt WHERE acknowledged_at IS NOT NULL", 1)
			checkRow("helper.db", "SELECT execution FROM deployment_execution WHERE job=? AND bundle=?", execution, job, binding)
			var wire []byte
			if err = central.db.QueryRow("SELECT receipt FROM terminal WHERE job=? AND execution=?", job, execution).Scan(&wire); err != nil {
				t.Fatal(err)
			}
			if unacked != "" && unacked != string(wire) {
				t.Fatal("changed original outbox bytes")
			}
			doc, e := signing.Verify("MC-04", wire, fixture.Scope, p.site.Config.Trust)
			if e != nil {
				t.Fatal(e)
			}
			state := "SUCCEEDED"
			table := "effect"
			if tc.wantEffects == 0 {
				state = "REJECTED"
				table = "no_effect"
			}
			body := doc["body"].(signing.Object)
			if body["state"] != state || body["execution_id"] != execution || body["job_id"] != job || body["observed_sha256"] != profile.Artifact {
				t.Fatal(body)
			}
			checkRow("agent.db", "SELECT signed_receipt FROM execution_receipt WHERE execution_id=?", string(wire), execution)
			checkRow(profile.Product+".db", "SELECT artifact FROM "+table+" WHERE job=? AND execution=?", profile.Artifact, job, execution)
			q := pl.Activation{Product: profile.Product, Job: job, Execution: execution, Artifact: profile.Artifact, Release: profile.Release, Expires: fixture.Now.Add(time.Hour).Format(time.RFC3339)}
			client := p.apps[profile.Product]
			observed, e := client.Lookup(ctx, q)
			if e != nil {
				t.Fatal(e)
			}
			duplicate, e := client.Apply(ctx, q)
			if e != nil || duplicate != observed {
				t.Fatal("delayed apply", observed, duplicate, e)
			}
			if tc.wantEffects == 0 && duplicate.Phase != "BLOCKED" {
				t.Fatal(duplicate)
			}
			if tc.wantEffects == 1 && duplicate.Phase != "APPLIED" {
				t.Fatal(duplicate)
			}
			other := q
			other.Job = "77777777-7777-4777-8777-777777777777"
			if _, e = client.Fence(ctx, other); e == nil {
				t.Fatal("fenced a different job with the original execution ID")
			}
			if _, e = client.Apply(ctx, other); e == nil {
				t.Fatal("applied a different job with the original execution ID")
			}
			q.Execution = "77777777-7777-4777-8777-777777777777"
			if _, e = client.Apply(ctx, q); e == nil {
				t.Fatal("changed execution accepted")
			}
			count(profile.Product+".db", "SELECT count(*) FROM effect", tc.wantEffects)
			count(profile.Product+".db", "SELECT count(*) FROM no_effect", 1-tc.wantEffects)
			if strings.Contains(tc.id, "ops-job") {
				sock := deployment.Socket{Endpoint: endpoint(hpath)}
				if r := helperReply(t, sock, "execute", b); r.Status != deployment.Denied {
					t.Fatal("revoked job executed", r)
				}
				if r := helperReply(t, sock, "execute", p.jobs[1]); r.Status != deployment.Denied {
					t.Fatal("new job used revoked authority", r)
				}
				changed := b
				changed.Manifest = p.jobs[1].Manifest
				if r := helperReply(t, sock, "recover", changed); r.Status != deployment.Denied {
					t.Fatal("substituted signed artifact", r)
				}
				if r := helperReply(t, sock, "recover", p.jobs[1]); r.Status != deployment.Denied {
					t.Fatal("unknown admission", r)
				}
				for _, field := range []string{"tenant_id", "site_id", "installation_id", "agent_id", "nonce"} {
					d := fixture.Doc(b.Job)
					body := d["body"].(signing.Object)
					if field == "tenant_id" || field == "site_id" {
						d[field] = "77777777-7777-4777-8777-777777777777"
					} else {
						body[field] = "77777777-7777-4777-8777-777777777777"
						if field == "nonce" {
							body[field] = "77777777777747778777777777777777"
						}
					}
					changed = b
					changed.Job = p.site.Sign(d, "mock-job")
					if r := helperReply(t, sock, "recover", changed); r.Status != deployment.Denied {
						t.Fatal(field, r)
					}
				}
				count("helper.db", "SELECT count(*) FROM consumed_grant", 1)
				count(profile.Product+".db", "SELECT count(*) FROM effect", tc.wantEffects)
			}
			result := helperReply(t, deployment.Socket{Endpoint: endpoint(hpath)}, "lookup", b)
			if result.Result == nil || result.Result.State != state || result.Result.Execution != execution {
				t.Fatal(result)
			}
			if result.Result.Product != profile.Product || result.Result.Artifact != profile.Artifact {
				t.Fatal("artifact lost")
			}
			t.Logf("%s job=%s execution=%s effects=%d grants(agent/helper)=1/1 receipts=1 state=%s", tc.id, job, execution, tc.wantEffects, state)
			h.kill(t)
			product.kill(t)
		})
	}
}
