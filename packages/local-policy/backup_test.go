package policy_test

import (
	"context"
	"strings"
	"testing"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/tests/fixtures"
)

func TestBackupAdmissionModes(t *testing.T) {
	for _, mode := range []string{"legacy", "connected"} {
		for _, fault := range []string{"valid", "missing policy", "other job", "wrong signer", "wrong artifact", "wrong product", "wrong manifest", "wrong categories", "signed age mismatch", "stale", "unencrypted", "failed integrity", "error state"} {
			t.Run(mode+"/"+fault, func(t *testing.T) {
				s := fixture.NewSite()
				b := s.Deployment(1, "product", "LOCAL_BUSY", "r2", "APP_CONFIG", "a")
				id := fixture.Doc(b.Job)["body"].(signing.Object)["job_id"].(string)
				cfg := s.Config.Clone()
				req := cfg.BackupForJob[id]
				doc := fixture.Doc(b.Backup)
				body := doc["body"].(signing.Object)
				switch fault {
				case "missing policy":
					cfg.BackupForJob = nil
				case "other job":
					b.Backup = s.Deployment(2, "product", "LOCAL_BUSY", "r2", "APP_CONFIG", "b").Backup
				case "wrong signer":
					req.Signer = "other"
				case "wrong artifact":
					req.Artifact = strings.Repeat("b", 64)
				case "wrong product":
					req.Product = "other"
				case "wrong manifest":
					req.BackupManifest = strings.Repeat("a", 64)
				case "wrong categories":
					req.Categories = []string{"MANAGEMENT_AGENT"}
				case "signed age mismatch":
					body["backup_age_seconds"] = 30
				case "stale":
					body["last_success_at"] = "2026-11-20T05:00:00Z"
				case "unencrypted":
					body["encrypted"] = false
				case "failed integrity":
					body["integrity_verified"] = false
				case "error state":
					body["error_state"] = "BACKUP_FAILED"
				}
				if cfg.BackupForJob != nil {
					cfg.BackupForJob[id] = req
				}
				if fault != "other job" {
					b.Backup = s.Sign(doc, "mock-backup")
				}
				preflight := false
				if mode == "connected" {
					cfg.LocalCheck = func(context.Context, string, []string) error { preflight = true; return nil }
				} else {
					cfg.Preflight = func(string, []string) bool { preflight = true; return true }
				}
				_, err := cfg.Authorize(b, fixture.Now)
				if (err == nil) != (fault == "valid") || preflight != (fault == "valid") {
					t.Fatalf("preflight=%v err=%v", preflight, err)
				}
			})
		}
	}
}
