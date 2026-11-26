package fixture

import (
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
	"zyposoft.local/operations/packages/local-policy"
	"zyposoft.local/operations/packages/signing-go"
)

var Now = time.Date(2026, 11, 24, 5, 1, 0, 0, time.UTC)

const Install = "33333333-3333-4333-8333-333333333333"
const AgentID = "44444444-4444-4444-8444-444444444444"

type Site struct {
	Config policy.Config
	priv   map[string]ed25519.PrivateKey
}

func NewSite() *Site {
	digest, err := policy.Digest(Example("MC-05"))
	if err != nil {
		panic(err)
	}
	s := &Site{Config: policy.Config{Scope: Scope, Installation: Install, Agent: AgentID, Version: "1.2.0", Trust: Trust(), TestArtifacts: map[string]string{digest: "zyposono"}, Preflight: func(_ string, _ []string) bool { return true }}, priv: map[string]ed25519.PrivateKey{}}
	for kid, purpose := range map[string]string{"mock-release": "release_signer", "mock-backup": "agent_backup_receipt", "mock-job": "ops_job", "rotated-job": "ops_job", "mock-receipt": "agent_receipt", "mock-grant": "ops_approval", "mock-local": "site_authoriser"} {
		seed := sha256.Sum256([]byte("PUBLIC deterministic TEST fixture only/" + kid))
		key := ed25519.NewKeyFromSeed(seed[:])
		s.priv[kid] = key
		rec := signing.Key{Alg: "Ed25519", Purpose: purpose, Public: key.Public()}
		if purpose == "agent_backup_receipt" || purpose == "agent_receipt" || purpose == "site_authoriser" {
			rec.Scope = Scope
		}
		s.Config.Trust[kid] = rec
	}
	// Independently provision the two legacy release jobs before opening either ledger.
	s.Config.BackupForJob = map[string]policy.BackupRequirement{}
	for _, seq := range []int{1, 2} {
		s.Config.BackupForJob[testID(fmt.Sprint(seq))] = policy.BackupRequirement{Product: "zyposono", Artifact: digest, BackupManifest: legacyBackupDigest(seq), Signer: "mock-backup", Categories: []string{"APP_CONFIG"}}
	}
	return s
}
func (s *Site) ReceiptKey() (string, crypto.PrivateKey, error) {
	return "mock-receipt", s.priv["mock-receipt"], nil
}
func Doc(raw []byte) signing.Object {
	var doc signing.Object
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	return doc
}
func JSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
func (s *Site) Sign(doc signing.Object, kid string) []byte {
	delete(doc, "signature")
	id := doc["contract_id"].(string)
	sc := signing.Scope{}
	if tenant, ok := doc["tenant_id"].(string); ok {
		sc = signing.Scope{Tenant: tenant, Site: doc["site_id"].(string)}
	}
	wire, err := signing.Sign(id, JSON(doc), sc, s.Config.Trust, kid, s.priv[kid])
	if err != nil {
		panic(err)
	}
	return wire
}
func testID(label string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fixture/"+label)).String()
}
func (s *Site) Job(seq int, deploy bool, grants []string, kid string) policy.Bundle {
	doc := Doc(Example("MC-03"))
	body := doc["body"].(signing.Object)
	id := testID(fmt.Sprint(seq))
	body["job_id"] = id
	body["idempotency_key"] = id
	body["nonce"] = strings.ReplaceAll(id, "-", "")
	body["sequence"] = seq
	body["approval_grant_ids"] = []string{}
	b := policy.Bundle{}
	if !deploy {
		body["command_type"] = "REQUEST_HEALTH"
		body["params"] = signing.Object{}
	} else {
		b.Manifest = Example("MC-05")
		backup := Doc(Example("MC-14"))
		e := backup["body"].(signing.Object)
		e["backup_age_seconds"] = 0
		e["backup_manifest_sha256"] = legacyBackupDigest(seq)
		b.Backup = s.Sign(backup, "mock-backup")
		ids := []string{}
		for _, label := range grants {
			grant := Doc(Example("MC-09"))
			g := grant["body"].(signing.Object)
			gid := testID(label)
			g["grant_id"] = gid
			ids = append(ids, gid)
			g["target_digest_sha256"] = body["params"].(signing.Object)["manifest_sha256"]
			sig, err := signing.SignAttestation("MC-09", JSON(grant), s.Config.Trust, "mock-local", s.priv["mock-local"])
			if err != nil {
				panic(err)
			}
			g["site_authoriser_attestation"] = sig
			b.Grants = append(b.Grants, s.Sign(grant, "mock-grant"))
		}
		body["approval_grant_ids"] = ids
	}
	b.Job = s.Sign(doc, kid)
	return b
}

func legacyBackupDigest(seq int) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("fixture legacy backup %d", seq))))
}
