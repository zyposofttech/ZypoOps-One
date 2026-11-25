//go:build linux

package sprint2

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/services/site-agent"
	"zyposoft.local/operations/tests/fixtures"
)

func TestConflictingSubmissions(t *testing.T) {
	for _, field := range []string{"bundle", "idempotency_key", "nonce", "sequence", "cross-product grant"} {
		t.Run(field, func(t *testing.T) {
			dir := testDir(t)
			p := makePlan(dir, true)
			startApps(t, dir, p)
			bad := p.jobs[1]
			original := fixture.Doc(p.jobs[0].Job)["body"].(signing.Object)
			switch field {
			case "bundle":
				bad = p.jobs[0]
				j := fixture.Doc(bad.Job)
				j["body"].(signing.Object)["expires_at"] = "2027-08-01T05:00:00Z"
				bad.Job = p.site.Sign(j, "mock-job")
			case "cross-product grant":
				profile := p.profiles[1]
				bad = p.site.Deployment(2, profile.Product, profile.Lock, profile.Release, "MANAGEMENT_AGENT", p.profiles[0].Product)
			default:
				j := fixture.Doc(bad.Job)
				j["body"].(signing.Object)[field] = original[field]
				bad.Job = p.site.Sign(j, "mock-job")
			}
			socket := serveHelper(t, dir, p)
			a, err := agent.OpenConnected(filepath.Join(dir, "agent.db"), ident(), p.site.Config, p.site, nil, makePlan(dir, false).apps, socket)
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			if _, err = a.Handle(context.Background(), p.jobs[0], clock()); err != nil {
				t.Fatal(err)
			}
			if _, err = a.Handle(context.Background(), bad, clock()); err == nil {
				t.Fatal("agent admitted conflict")
			}
			r := helperReply(t, socket, "execute", bad)
			if r.Status != "REFUSED" && r.Status != "DENIED" {
				t.Fatal(r)
			}
			for _, q := range []struct {
				file, table string
				want        int
			}{
				{"agent.db", "approval_consumption", 1}, {"helper.db", "consumed_grant", 1},
				{p.profiles[0].Product + ".db", "effect", 1}, {p.profiles[1].Product + ".db", "effect", 0},
			} {
				if n := scalar(t, filepath.Join(dir, q.file), "SELECT count(*) FROM "+q.table); n != q.want {
					t.Fatal(q, n)
				}
			}
			if _, err = a.Handle(context.Background(), p.jobs[1], clock()); err != nil {
				t.Fatal("valid next authorization lost", err)
			}
		})
	}
}
func TestCompetingOperations(t *testing.T) {
	for _, first := range []string{"abort", "execute", "concurrent"} {
		t.Run(first, func(t *testing.T) {
			dir := testDir(t)
			p := makePlan(dir, true)
			startApps(t, dir, p)
			s := serveHelper(t, dir, p)
			b := p.jobs[0]
			if first != "concurrent" {
				helperReply(t, s, first, b)
			}
			start := make(chan struct{})
			var wg sync.WaitGroup
			for _, op := range []string{"lookup", "execute", "abort", "execute", "lookup"} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					_, err := s.Exchange(context.Background(), deployment.Request{Operation: op, Bundle: b})
					if err != nil {
						t.Error(err)
					}
				}()
			}
			close(start)
			wg.Wait()
			final := helperReply(t, s, "execute", b)
			effects := 0
			switch final.Status {
			case "DONE":
				if first == "abort" || final.Result.State != "SUCCEEDED" {
					t.Fatal(final)
				}
				effects = 1
			case "REFUSED":
				if first == "execute" {
					t.Fatal("completed execution overwritten")
				}
			default:
				t.Fatal(final)
			}
			if n := scalar(t, filepath.Join(dir, p.profiles[0].Product+".db"), "SELECT count(*) FROM effect"); n != effects {
				t.Fatal(n)
			}
			if n := scalar(t, filepath.Join(dir, "helper.db"), "SELECT count(*) FROM consumed_grant"); n != effects {
				t.Fatal(n)
			}
		})
	}
}

type substitutedResult struct {
	deployment.Socket
	foreign string
}

func (s substitutedResult) Exchange(ctx context.Context, q deployment.Request) (deployment.Reply, error) {
	r, err := s.Socket.Exchange(ctx, q)
	if err == nil && r.Result != nil {
		r.Result.Execution = s.foreign
	}
	return r, err
}
func TestSubstitutedHelperExecution(t *testing.T) {
	dir := testDir(t)
	p := makePlan(dir, true)
	startApps(t, dir, p)
	s := serveHelper(t, dir, p)
	path := filepath.Join(dir, "agent.db")
	open := func(h deployment.Helper) *agent.Agent {
		a, err := agent.OpenConnected(path, ident(), p.site.Config, p.site, nil, makePlan(dir, false).apps, h)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	foreign, err := deployment.BundleDigest(p.jobs[1])
	if err != nil {
		t.Fatal(err)
	}
	a := open(substitutedResult{s, deployment.ExecutionID(foreign)})
	if _, err := a.Handle(context.Background(), p.jobs[0], clock()); err == nil {
		t.Fatal("signed substituted execution")
	}
	a.Close()
	if n := scalar(t, path, "SELECT count(*) FROM execution_receipt"); n != 0 {
		t.Fatal(n)
	}
	a = open(s)
	defer a.Close()
	if _, err := a.Handle(context.Background(), p.jobs[0], clock()); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, filepath.Join(dir, p.profiles[0].Product+".db"), "SELECT count(*) FROM effect"); n != 1 {
		t.Fatal(n)
	}
}
