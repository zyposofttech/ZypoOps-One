package sprint2

import (
	"path/filepath"
	"testing"
	"zyposoft.local/operations/packages/deployment"
	"zyposoft.local/operations/packages/local-policy"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/tests/fixtures"
)

func TestCentralReceiptBinding(t *testing.T) {
	site := fixture.NewSite()
	b := site.Deployment(1, "product-test", "LOCAL_BUSY", "test-2", "APP_CONFIG", "g")
	path := filepath.Join(t.TempDir(), "central.db")
	alternate := site.Config.Trust["rotated-job"]
	alternate.Scope = fixture.Scope
	alternate.Purpose = "agent_receipt"
	site.Config.Trust["rotated-job"] = alternate
	c, err := OpenCentral(path, fixture.Scope, site.Config.Trust, primaryEnrolment())
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Submit(b); err != nil {
		t.Fatal(err)
	}
	if err = c.Submit(b); err != nil {
		t.Fatal(err)
	}
	changed := b
	changed.Backup = nil
	if err = c.Submit(changed); err == nil {
		t.Fatal("queue accepted conflicting bundle")
	}
	doc := fixture.Doc(fixture.Example("MC-04"))
	body := doc["body"].(signing.Object)
	body["job_id"] = fixture.Doc(b.Job)["body"].(signing.Object)["job_id"]
	artifact, _ := policy.Digest(b.Manifest)
	body["observed_sha256"] = artifact
	binding, err := deployment.BundleDigest(b)
	if err != nil {
		t.Fatal(err)
	}
	body["execution_id"] = deployment.ExecutionID(binding)
	body["stage"] = "VERIFIED"
	wire := site.Sign(doc, "mock-receipt")
	other := fixture.NewSite()
	for _, field := range []string{"tenant_id", "site_id"} {
		changed := fixture.Doc(wire)
		changed[field] = "77777777-7777-4777-8777-777777777777"
		key := other.Config.Trust["mock-receipt"]
		key.Scope = signing.Scope{Tenant: changed["tenant_id"].(string), Site: changed["site_id"].(string)}
		other.Config.Trust["mock-receipt"] = key
		if err := c.accept(other.Sign(changed, "mock-receipt")); err == nil {
			t.Fatal(field)
		}
	}
	if err := c.accept(site.Sign(fixture.Doc(wire), "rotated-job")); err == nil {
		t.Fatal("accepted other enrolled signer")
	}
	for field, value := range map[string]string{"job_id": "77777777-7777-4777-8777-777777777777", "execution_id": "77777777-7777-4777-8777-777777777777", "observed_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "stage": "POST_VERIFY", "state": "FAILED", "error_code": "PRECONDITION_FAILED", "clock_confidence": "UNTRUSTED"} {
		changed := fixture.Doc(wire)
		changed["body"].(signing.Object)[field] = value
		if err := c.accept(site.Sign(changed, "mock-receipt")); err == nil {
			t.Fatal("accepted mismatched first receipt", field)
		}
	}
	for _, field := range []string{"installation_id", "agent_id"} {
		j := fixture.Doc(b.Job)
		j["body"].(signing.Object)[field] = "77777777-7777-4777-8777-777777777777"
		other := b
		other.Job = site.Sign(j, "mock-job")
		if err := c.Submit(other); err == nil || err.Error() != "INVALID_JOB" {
			t.Fatal("target identity", field, err)
		}
	}
	if err = c.accept(wire); err != nil {
		t.Fatal(err)
	}
	legacy := "77777777-7777-4777-8777-777777777777"
	if err := c.restoreLegacyReservation(body["job_id"].(string), binding, legacy); err == nil {
		t.Fatal("changed assigned ID")
	}
	c.Close()
	c, err = OpenCentral(path, fixture.Scope, site.Config.Trust)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.accept(wire); err != nil {
		t.Fatal(err)
	}
	body["execution_id"] = "77777777-7777-4777-8777-777777777777"
	if err = c.accept(site.Sign(doc, "mock-receipt")); err == nil {
		t.Fatal("conflicting signed outcome accepted")
	}
	body["observed_sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err = c.accept(site.Sign(doc, "mock-receipt")); err == nil {
		t.Fatal("wrong artifact accepted")
	}
	var n int
	if err = c.db.QueryRow("SELECT count(*) FROM terminal").Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	// Seed an outstanding command in the old storage shape, before ID pins.
	next := site.Deployment(2, "product-test", "LOCAL_BUSY", "test-2", "APP_CONFIG", "next")
	digest, err := deployment.BundleDigest(next)
	if err != nil {
		t.Fatal(err)
	}
	id := fixture.Doc(next.Job)["body"].(signing.Object)["job_id"].(string)
	if _, err = c.db.Exec("INSERT INTO commands VALUES(?,?,?,?,?)", id, 2, digest, artifact, fixture.JSON(next)); err != nil {
		t.Fatal(err)
	}
	legacyDoc := fixture.Doc(wire)
	legacyBody := legacyDoc["body"].(signing.Object)
	legacyBody["job_id"] = id
	legacyBody["execution_id"] = legacy
	legacyWire := site.Sign(legacyDoc, "mock-receipt")
	if err = c.accept(legacyWire); err == nil {
		t.Fatal("accepted unprovisioned historical ID")
	}
	if err = c.restoreLegacyReservation(id, digest, legacy); err != nil {
		t.Fatal(err)
	}
	if err = c.accept(legacyWire); err != nil {
		t.Fatal(err)
	}
}

func primaryEnrolment() Enrolment {
	return Enrolment{fixture.Scope.Tenant, fixture.Scope.Site, fixture.Install, fixture.AgentID, "mock-receipt"}
}

func TestMultiSiteEnrolment(t *testing.T) {
	site := fixture.NewSite()
	path := filepath.Join(t.TempDir(), "central.db")
	first := primaryEnrolment()
	second := Enrolment{"88888888-8888-4888-8888-888888888888", "99999999-9999-4999-8999-999999999999", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "rotated-job"}
	otherScope := signing.Scope{Tenant: second.Tenant, Site: second.Site}
	key := site.Config.Trust[second.ReceiptKey]
	key.Scope = otherScope
	key.Purpose = "agent_receipt"
	site.Config.Trust[second.ReceiptKey] = key
	a, e := OpenCentral(path, fixture.Scope, site.Config.Trust, first, second)
	if e != nil {
		t.Fatal(e)
	}
	a.Close()
	for _, r := range []Enrolment{first, {first.Tenant, first.Site, second.Installation, first.Agent, first.ReceiptKey}} {
		if db, err := OpenCentral(path, fixture.Scope, site.Config.Trust, r); err == nil {
			db.Close()
			t.Fatal("conflicting enrolment")
		}
	}
	a, e = OpenCentral(path, fixture.Scope, site.Config.Trust)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := OpenCentral(path, otherScope, site.Config.Trust)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	var wires [][]byte
	for i, r := range []Enrolment{first, second} {
		c := []*Central{a, b}[i]
		bundle := site.Deployment(i+1, "product", "LOCAL_BUSY", "r2", "APP_CONFIG", "auth")
		j := fixture.Doc(bundle.Job)
		j["tenant_id"] = r.Tenant
		j["site_id"] = r.Site
		body := j["body"].(signing.Object)
		body["installation_id"] = r.Installation
		body["agent_id"] = r.Agent
		body["sequence"] = 1
		bundle.Job = site.Sign(j, "mock-job")
		if e = c.Submit(bundle); e != nil {
			t.Fatal(e)
		}
		other := []*Central{b, a}[i]
		if other.Submit(bundle) == nil {
			t.Fatal("foreign job")
		}
		receipt := fixture.Doc(fixture.Example("MC-04"))
		receipt["tenant_id"] = r.Tenant
		receipt["site_id"] = r.Site
		out := receipt["body"].(signing.Object)
		out["job_id"] = body["job_id"]
		out["stage"] = "VERIFIED"
		binding, e := deployment.BundleDigest(bundle)
		if e != nil {
			t.Fatal(e)
		}
		out["execution_id"] = deployment.ExecutionID(binding)
		out["observed_sha256"] = body["params"].(signing.Object)["manifest_sha256"]
		wire := site.Sign(receipt, r.ReceiptKey)
		wires = append(wires, wire)
		if other.accept(wire) == nil {
			t.Fatal("foreign signer")
		}
	}
	for i, c := range []*Central{a, b} {
		if err := c.accept(wires[i]); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := a.db.QueryRow("SELECT count(*) FROM terminal").Scan(&n); err != nil || n != 2 {
		t.Fatal(n, err)
	}
}
