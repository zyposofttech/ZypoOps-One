package cryptography

import (
	"encoding/base64"
	"strings"
	"testing"
	contracts "zyposoft.local/operations/packages/management-contracts"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/tests/fixtures"
)

func TestGoRejections(t *testing.T) {
	for _, name := range []string{"revoked", "unknown", "wrong purpose", "alg substitution", "DER length", "absent signature", "tamper", "unknown version", "wrong site"} {
		t.Run(name, func(t *testing.T) {
			doc := fixture.Doc(fixture.Example("MC-04"))
			sig := doc["signature"].(signing.Object)
			keys := fixture.Trust()
			kid := sig["kid"].(string)
			rec := keys[kid]
			scope := fixture.Scope
			switch name {
			case "revoked":
				rec.Revoked = true
				keys[kid] = rec
			case "unknown":
				delete(keys, kid)
			case "wrong purpose":
				rec.Purpose = "ops_job"
				keys[kid] = rec
			case "alg substitution":
				sig["alg"] = "Ed25519"
			case "DER length":
				sig["value"] = base64.RawURLEncoding.EncodeToString(make([]byte, 70))
			case "absent signature":
				delete(doc, "signature")
			case "tamper":
				doc["body"].(signing.Object)["stage"] = "MODIFIED"
			case "unknown version":
				doc["schema_version"] = "1.3.0"
			case "wrong site":
				scope.Site = "different"
			}
			if _, err := signing.Verify("MC-04", fixture.JSON(doc), scope, keys); err == nil {
				t.Fatal("invalid accepted")
			}
		})
	}
}
func TestHygieneBoundaries(t *testing.T) {
	if _, _, err := contracts.Decode("LOCAL-01", []byte(`{"unknown":true}`)); err == nil {
		t.Fatal("root accepted unknown")
	}
	if _, _, err := contracts.Decode("LOCAL-01/readiness_response", []byte(`{"ready":true,"service_codes":["APP_READY"],"version":"1.5"}`)); err != nil {
		t.Fatal(err)
	}
	doc := fixture.Doc(fixture.Example("MC-10"))
	for _, n := range []int{128, 129} {
		doc["body"].(signing.Object)["clinical_approval_ref"] = strings.Repeat("a", n)
		_, _, err := contracts.Decode("MC-10", fixture.JSON(doc))
		if (n == 128) != (err == nil) {
			t.Fatal(n, err)
		}
	}
}
