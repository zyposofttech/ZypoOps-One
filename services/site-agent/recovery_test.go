package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"zyposoft.local/operations/tests/fixtures"
)

func TestRebootAndAuthenticatedClockRecovery(t *testing.T) {
	for _, cause := range []string{"ordinary reboot", "wall-clock rollback", "monotonic regression"} {
		t.Run(cause, func(t *testing.T) {
			site := fixture.NewSite()
			db := filepath.Join(t.TempDir(), "clock.db")
			now := fixture.Now
			var served atomic.Int64
			served.Store(now.Unix())
			peer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Date", time.Unix(served.Load(), 0).UTC().Format(http.TimeFormat))
				w.WriteHeader(http.StatusOK)
			}))
			defer peer.Close()
			source, err := NewTestTimeSource(peer.URL, peer.Certificate())
			if err != nil {
				t.Fatal(err)
			}
			id := Identity{fixture.Scope.Tenant, fixture.Scope.Site, fixture.Install, fixture.AgentID, strings.Repeat("a", 64)}
			open := func(source *TestTimeSource) *Agent {
				a, err := OpenWithTestTimeSource(db, id, site.Config, site, nil, source)
				if err != nil {
					t.Fatal(err)
				}
				return a
			}
			a := open(source)
			ctx := context.Background()
			first := sample()
			if _, err = a.Handle(ctx, site.Job(1, false, nil, "mock-job"), first); err != nil {
				t.Fatal(err)
			}
			a.Close()
			a = open(source)
			observed := first
			switch cause {
			case "ordinary reboot":
				observed.Boot = "boot-B"
				observed.Monotonic = 0
				observed.UTC = now.Add(time.Minute)
			case "wall-clock rollback":
				observed.UTC = now.Add(-time.Hour)
			default:
				observed.Monotonic--
			}
			next := site.Job(2, false, nil, "mock-job")
			if _, err = a.Handle(ctx, next, observed); err == nil || err.Error() != "CLOCK_UNTRUSTED" {
				t.Fatal("unsafe clock accepted", err)
			}
			a.Close()
			// A different locally configured trust root cannot authenticate this TLS server.
			unrelated := unrelatedCertificate(t)
			wrong, err := NewTestTimeSource(peer.URL, unrelated)
			if err != nil {
				t.Fatal(err)
			}
			a = open(wrong)
			corrected := observed
			corrected.UTC = now.Add(2 * time.Minute)
			corrected.Monotonic = first.Monotonic + 1
			if err = a.RecoverClock(ctx, corrected); err == nil {
				t.Fatal("untrusted clock authority accepted")
			}
			a.Close()
			a = open(source)
			defer a.Close()
			if _, err = a.Handle(ctx, next, corrected); err == nil {
				t.Fatal("restart reset clock confidence")
			}
			badBoot := corrected
			badBoot.Boot = "unobserved"
			if err = a.RecoverClock(ctx, badBoot); err == nil {
				t.Fatal("unexpected boot accepted")
			}
			if err = a.RecoverClock(ctx, corrected); err == nil {
				t.Fatal("stale time proof accepted")
			}
			served.Store(corrected.UTC.Unix())
			if err = a.RecoverClock(ctx, corrected); err != nil {
				t.Fatal(err)
			}
			a.Close()
			a = open(source)
			defer a.Close()
			if _, err = a.Handle(ctx, next, corrected); err != nil {
				t.Fatal(err)
			}
			if err = a.RecoverClock(ctx, corrected); err == nil {
				t.Fatal("replayed recovery accepted")
			}
		})
	}
}
func TestRejectedJobPreservesClockWatermark(t *testing.T) {
	site := fixture.NewSite()
	path := filepath.Join(t.TempDir(), "agent.db")
	a := openAgent(t, path, site)
	next := site.Job(1, false, nil, "mock-job")
	future := sample()
	untrusted := sample()
	untrusted.UTC = untrusted.UTC.AddDate(20, 0, 0)
	untrusted.Trusted = false
	if _, err := a.Handle(context.Background(), next, untrusted); err == nil || err.Error() != "CLOCK_UNTRUSTED" {
		t.Fatal(err)
	}
	future.UTC = future.UTC.AddDate(2, 0, 0)
	if _, err := a.Handle(context.Background(), next, future); err == nil || err.Error() != "JOB_EXPIRED" {
		t.Fatal(err)
	}
	a.Close()
	a = openAgent(t, path, site)
	defer a.Close()
	if _, err := a.Handle(context.Background(), next, sample()); err == nil || err.Error() != "CLOCK_UNTRUSTED" {
		t.Fatal("expiry rollback", err)
	}
}

func unrelatedCertificate(t *testing.T) *x509.Certificate {
	t.Helper()
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := x509.Certificate{SerialNumber: big.NewInt(7), NotBefore: fixture.Now.Add(-time.Hour), NotAfter: fixture.Now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, &cert, &cert, pub, private)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
