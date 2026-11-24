package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

// TestTimeSource accepts only a directly connected loopback HTTPS peer with an
// explicitly supplied test certificate. No system trust or management command can select a source.
type TestTimeSource struct {
	endpoint    string
	certificate *x509.Certificate
}

func NewTestTimeSource(endpoint string, certificate *x509.Certificate) (*TestTimeSource, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || certificate == nil {
		return nil, errors.New("TIME_SOURCE_CONFIG")
	}
	addr, err := netip.ParseAddr(u.Hostname())
	if err != nil || !addr.IsLoopback() {
		return nil, errors.New("TIME_SOURCE_CONFIG")
	}
	pin, err := x509.ParseCertificate(append([]byte(nil), certificate.Raw...))
	if err != nil {
		return nil, errors.New("TIME_SOURCE_CONFIG")
	}
	return &TestTimeSource{u.String(), pin}, nil
}
func (s *TestTimeSource) read(ctx context.Context, floor time.Time) (time.Time, error) {
	roots := x509.NewCertPool()
	roots.AddCert(s.certificate)
	client := http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport: &http.Transport{DisableKeepAlives: true, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13,
			// Verify at the persisted trusted time, then check certificate validity again at the recovered instant.
			Time: func() time.Time { return floor },
			VerifyConnection: func(cs tls.ConnectionState) error {
				if !cs.PeerCertificates[0].Equal(s.certificate) {
					return errors.New("TIME_PEER")
				}
				return nil
			}}}}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.endpoint, nil)
	if err != nil {
		return time.Time{}, err
	}
	response, err := client.Do(req)
	if err != nil {
		return time.Time{}, errors.New("TIME_PEER")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return time.Time{}, errors.New("TIME_SOURCE_UNAVAILABLE")
	}
	instant, err := http.ParseTime(response.Header.Get("Date"))
	if err != nil || instant.Before(s.certificate.NotBefore) || instant.After(s.certificate.NotAfter) {
		return time.Time{}, errors.New("TIME_PROOF_INVALID")
	}
	return instant, nil
}

// RecoverClock is a local adapter boundary. It is never invoked by an MC-03 operation.
// It binds a fresh authenticated reading to the observed boot and preserves the durable floor.
func (a *Agent) RecoverClock(ctx context.Context, s Sample) error {
	if a.recovery == nil || s.Boot == "" || s.Monotonic < 0 || !s.Trusted {
		return errors.New("TIME_RECOVERY_UNAVAILABLE")
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var watermark, boot, confidence string
	var tick int64
	err = tx.QueryRowContext(ctx, "SELECT last_trusted_utc,boot_id,last_monotonic_ns,clock_confidence FROM clock_watermark WHERE installation_id=?", a.config.Installation).Scan(&watermark, &boot, &tick, &confidence)
	if err != nil {
		return err
	}
	floor, err := time.Parse(time.RFC3339Nano, watermark)
	if err != nil {
		return err
	}
	if confidence != "UNTRUSTED" || s.Boot != boot || s.Monotonic < tick || s.UTC.Before(floor) {
		return errors.New("TIME_RECOVERY_CONTEXT")
	}
	instant, err := a.recovery.read(ctx, floor)
	if err != nil {
		return err
	}
	// HTTP Date has second precision. The two-second agreement bound is test policy.
	if instant.Before(floor.Truncate(time.Second)) || s.UTC.Sub(instant).Abs() > 2*time.Second {
		return errors.New("TIME_PROOF_STALE")
	}
	_, err = tx.ExecContext(ctx, "UPDATE clock_watermark SET last_trusted_utc=?,last_monotonic_ns=?,clock_confidence='TRUSTED' WHERE installation_id=?", s.UTC.UTC().Format(time.RFC3339Nano), s.Monotonic, a.config.Installation)
	if err != nil {
		return err
	}
	return tx.Commit()
}
