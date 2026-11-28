# ZypoOps One — O1 Sprint 2R Final Independent Acceptance Re-Audit

**Acceptance Review Date:** 26 November 2026  
**Repository:** `https://github.com/zyposofttech/ZypoOps-One` (branch `main`)  
**Original Audited Baseline Commit:** `8f9d88be2013c9b014202fc2435ec1368d83575c` (`O1-Sprint 2 Completed`)  
**Remediation Implementation Commit:** `713e9240638d4fe484ff539bb2b1dc0e0ec885a5`  
**Latest Evidence Commit Audited:** `92b3b4059dc96ea4ec78222d6774f4ad16873c23` (no runtime source changes from `713e9240638d4fe484ff539bb2b1dc0e0ec885a5`)  
**Verified Candidate Source SHA-256:** `3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6`  
**Audit Scope:** Focused independent acceptance review of the 8 Sprint 2 audit findings (`AUD-S2-P1-01` through `AUD-S2-P3-03`) and Sprint 3 test-only entry gate  
**Audit Mode:** Read-only independent acceptance review (no source modifications, no commits/pushes, no Sprint 3 implementation)

---

## 1. Executive Acceptance Verdict

- **O1 Sprint 2R Final Engineering Verdict (Test-Only Scope):** **`GO_TEST_ONLY`**
- **Ready for O1 Sprint 3 (Test-Only):** **`YES_TEST_ONLY`**
- **Original Sprint 2 Findings Closed:** **8 / 8** (`0` open, `0` regressions)
- **New Technical Findings Introduced:** **0** (`0 P0`, `0 P1`, `0 P2`, `0 P3`)
- **Overall Engineering Score:** **9.2 / 10** (upgraded from 8.5 / 10)
- **O0 Production Gate:** **`NO_GO`** (`0/8` human approvals on `OD01–OD08`; `Q01–Q07` open; joint `LOCAL-01` sign-off open)
- **Clinical Production Readiness:** **`NO_GO`**

All eight technical findings from the independent Sprint 2 audit (`2 P1`, `3 P2`, `3 P3`) have been cleanly, soundly, and completely remediated without introducing any authorization bypass, concurrency regression, or governance erosion. All three hosted GitHub Actions CI runs (`45594545137`, `45768586579`, `45768899686`) passed, the candidate source fingerprint (`3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6`) and all **42/42** evidence files passed independent SHA-256 verification, and line-by-line adversarial source review confirmed every security invariant. **Development may safely proceed to O1 Sprint 3 strictly within the authorised test-only scope.**

---

## 2. Candidate Identity, Source Fingerprint & Evidence Integrity

| Check | Observed Value / Outcome | Verification Basis |
|---|---|---|
| Latest Evidence Commit | `92b3b4059dc96ea4ec78222d6774f4ad16873c23` (snapshot matches remediation source `713e9240638d4fe484ff539bb2b1dc0e0ec885a5`) | Independent archive inspection |
| Candidate Source Fingerprint | `3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6` | Independently executed `python3 scripts/verify-sprint2r-evidence.py` — **EXACT MATCH** |
| Evidence Register Integrity | `42 / 42` registered log files verified byte-for-byte against `docs/audit/sprint-2r-evidence/index.json` | Independently executed `python3 scripts/verify-sprint2r-evidence.py` — **0 MISMATCHES** |
| Original Audit Report Integrity | `docs/audit/O1-Sprint-2-Independent-Audit.md` SHA-256 = `b0e5d99bc3c6d7d26b01065e2be2d192795ba4727bbb36be321352a986eb2df6` | Independently executed `sha256sum` — **EXACT MATCH** |
| Hosted CI — Baseline (`8f9d88be...`) | Run `45594545137` — **PASS** (11/11 workflow steps including Linux Go `-race` & PG16 RLS) | Connected GitHub verification + archived `hosted-baseline.log` |
| Hosted CI — Remediation (`713e9240...`) | Run `45768586579` — **PASS** | Connected GitHub verification |
| Hosted CI — Evidence (`92b3b405...`) | Run `45768899686` — **PASS** | Connected GitHub verification |
| Contract Binding Drift | `scripts/generate-types.mjs` vs `types.ts` and `types_generated.go` — **ZERO DIFF** | Independently executed in audit sandbox (`Node v18.20.4`) |

---

## 3. Eight Original Finding Dispositions

| Finding ID | Severity | Disposition | Key Source References | Evidence Basis |
|---|---|---|---|---|
| **`AUD-S2-P1-01`** | P1 | **`CLOSED_EVIDENCE_ACCEPTED`** | `services/site-agent/agent.go:178-191,350-360`; `services/site-agent/deployment.go:155-316`; `services/local-helper/deployment.go:46-124,247-264`; `tests/sprint2/crash_linux_test.go:92-100` | Adversarial source audit + `red-linux.log` baseline failure (`UNTRUSTED_KEY`) + `linux-sprint2.log` & `linux-race-sprint2.log` (21/21 `TestCrashMatrix` subcases passing) + hosted CI `45768586579` |
| **`AUD-S2-P1-02`** | P1 | **`CLOSED_EVIDENCE_ACCEPTED`** | `.github/workflows/foundation.yml:23-38`; `scripts/verify-sprint2r-evidence.py`; `docs/audit/sprint-2r-evidence/index.json` | All 3 hosted CI runs passed + 42/42 evidence files & source fingerprint independently verified + workflow retains `foundation-evidence-<sha>` artifacts |
| **`AUD-S2-P2-01`** | P2 | **`CLOSED_EVIDENCE_ACCEPTED`** | `services/local-helper/socket_linux.go:19-56`; `services/local-helper/helper.go:160-179`; `services/local-helper/deployment.go:67-71,171,200`; `tests/sprint2/remediation_linux_test.go:122-194` | Source audit + `linux-sprint2.log` & `linux-race-sprint2.log` (`TestOverlappingAdmissions`, `TestBoundedStalledConnections`, `TestConcurrentDuplicates`, `TestCompetingOperations`) + hosted CI |
| **`AUD-S2-P2-02`** | P2 | **`CLOSED_EVIDENCE_ACCEPTED`** | `packages/local-policy/policy.go:89-112`; `services/site-agent/agent.go:157-198`; `services/site-agent/deployment.go:39-134`; `services/local-helper/helper.go:81-129`; `services/local-helper/deployment.go:174-216` | Source audit + `linux-sprint2.log` & `linux-race-sprint2.log` (8 `TestPreflightTransactions` cases + 14 `TestEffectBoundaries` cases) + hosted CI |
| **`AUD-S2-P2-03`** | P2 | **`CLOSED_EVIDENCE_ACCEPTED`** | `packages/local-policy/policy.go:178-199`; `packages/local-policy/backup_test.go:11-67`; `tests/fixtures/site.go` | Source audit + `linux-local-policy.log` & `linux-race-local-policy.log` (27/27 `TestBackupAdmissionModes`) + 54 `TestAdmissionFailures` + hosted CI |
| **`AUD-S2-P3-01`** | P3 | **`CLOSED_INDEPENDENTLY_VERIFIED`** | `README.md:9,55,66`; `docs/implementation/O1-Sprint-2.md` (dated erratum); `docs/implementation/O1-Sprint-2-Checkpoint.md` (dated erratum) | Direct independent inspection confirming accurate status, preserved historical reports, and BRD v1.2 footer erratum |
| **`AUD-S2-P3-02`** | P3 | **`CLOSED_INDEPENDENTLY_VERIFIED`** | `services/site-agent/deployment.go:29-31,287-292`; `tests/sprint2/remediation_linux_test.go:318-376` | Source audit + independently executed Python `sqlite3` idempotent migration & receipt-immutability probe + `TestUnknownStageMigration` in Linux ordinary & `-race` logs |
| **`AUD-S2-P3-03`** | P3 | **`CLOSED_EVIDENCE_ACCEPTED`** | `tests/sprint2/central.go:22-227`; `tests/sprint2/central_test.go:131-212` | Source audit + `go-windows.log`, `race-windows.log`, `linux-sprint2.log`, `linux-race-sprint2.log` (`TestMultiSiteEnrolment`, `TestCentralReceiptBinding`) + hosted CI |

---

## 4. Deep Security Verification of High-Risk Remediations

### `AUD-S2-P1-01` — Revoked Signing-Key Reconciliation (Zero Authorization Bypass Verified)
Adversarial line-by-line review of `services/site-agent/agent.go`, `services/site-agent/deployment.go`, and `services/local-helper/deployment.go` confirmed every mandatory security invariant:
1. **Fresh deliveries require live valid authority:** `Agent.handle()` (`agent.go:180`) calls `signing.Verify("MC-03", b.Job, a.config.Scope, a.config.Trust)` unconditionally before admission; `Helper.dispatch()` (`local-helper/deployment.go:46-50`) immediately returns `DENIED` whenever `liveErr != nil` for `"execute"` or `"abort"`.
2. **Historical recovery strictly bound to independently persisted admission records:**
   - Agent `resume()` (`site-agent/deployment.go:155-207`) is reachable only from the local `pending_deployment` queue (`j.state='RUNNING'`), joins `pending_deployment`, `received_job`, and `local_identity` on exact `job_id`, `p.binding`, `j.payload_sha256`, `j.issuer_kid`, `command_type='DEPLOY_RELEASE'`, `verified_at IS NOT NULL`, `installation_id`, `site_id`, `tenant_id`, and `agent_id`, and verifies every grant in `b.Grants` against `approval_consumption(job_id, grant_id, grant_sha256)`.
   - Helper `dispatch()` (`local-helper/deployment.go:88-115`) requires `deployment_execution JOIN simulated_deployment USING(job) WHERE job=? AND bundle=? AND digest=?` plus every `grant_id` present in `consumed_grant`. Modified bundles, substituted execution IDs, swapped products/artifacts, or mismatched tenant/site identities return `DENIED`.
3. **Zero write/mutation authority under revoked keys:** When `liveErr != nil`, Agent sends `operation="recover"` (`site-agent/deployment.go:209-211`), and Helper `reconcile()` (`local-helper/deployment.go:247-264`) can **only** observe an already-committed product effect (`app.Lookup`) or durably fence an unapplied reservation (`app.Fence`) — it can never call `app.Apply()`.
4. **Agent-only admission, missing observation, and revoked receipt signer fail safely:** If Helper has no reservation (Agent-only admission), Helper returns `DENIED`, Agent `deliver()` (`site-agent/deployment.go:231-233`) rejects with `HISTORICAL_RECOVERY_REQUIRED`, and `resume()` records `deployment_recovery_hold` (`site-agent/deployment.go:215-216`), blocking all subsequent `admit()` calls (`site-agent/deployment.go:101-110`). Missing product observation preserves the hold; revoked `agent_receipt` key fails inside `a.finish()` and rolls back without emitting an untrusted receipt.
5. **Clock & replay integrity preserved:** Both `resume()` (`site-agent/deployment.go:198-207`) and `Helper.dispatch()` (`local-helper/deployment.go:78-85`) still enforce trusted-clock watermarks and rollback latches during historical recovery.

### `AUD-S2-P2-01` — Bounded Helper Concurrency
- `Helper.ServeWithLimit()` (`services/local-helper/socket_linux.go:25-56`) enforces `1 <= limit <= 256` (default 16), uses a non-blocking buffered semaphore `slots` so excess connections are closed immediately without spawning unbounded goroutines, and drains all workers cleanly via `cancel(); workers.Wait()` on shutdown.
- `Helper.lockJob()` (`services/local-helper/helper.go:162-179`) serializes same-job reconciliation with context cancellation and deletes map entries upon release (zero unbounded memory growth).
- `TestOverlappingAdmissions` and `TestBoundedStalledConnections` prove genuine overlapping product preflight barriers, stalled partial-frame coexistence, saturation rejection, and clean worker drain under both Linux ordinary and `-race` builds.

### `AUD-S2-P2-02` — SQLite Transaction Boundaries
- Both `Agent` (`agent.go:186`, `site-agent/deployment.go:39-134`) and `Helper` (`helper.go:81-129`, `local-helper/deployment.go:174-216`) persist the ingress clock watermark in a short transaction, release the SQLite writer completely before calling `AuthorizeContext` (live `LOCAL-01` HTTP I/O), and then re-acquire a short atomic `BEGIN IMMEDIATE` transaction that rechecks clock monotonicity, `Authorized.Current(now)` (minimum deadline across job, manifest, grants, and 24h backup window), and `siteHeld(tx)` before committing grant consumption and reservation.
- `TestPreflightTransactions` (8 role × fault cases) proves an independent SQLite writer acquires a write lock within 80 ms while `LOCAL-01` is paused, and verifies zero grant/effect leakage on insert failure, cancel, expiry, and clock rollback.

### `AUD-S2-P2-03`, `AUD-S2-P3-01`, `AUD-S2-P3-02`, `AUD-S2-P3-03`
- **MC-14 (`AUD-S2-P2-03`):** `policy.Validate()` (`packages/local-policy/policy.go:178-199`) enforces `BackupForJob` (signer, product, artifact, backup manifest digest, exact dataset categories, signed age/timestamp consistency, `< 24h` freshness, encryption, integrity, `error_state == "NONE"`) unconditionally across both legacy `Preflight` and connected `LocalCheck` modes. Verified by 26 `TestBackupAdmissionModes` subcases.
- **Documentation (`AUD-S2-P3-01`):** `README.md` accurately reflects committed Sprints 1–2 and Sprint 2R review status; dated errata appended to `O1-Sprint-2.md` and `O1-Sprint-2-Checkpoint.md` document commit `8f9d88be...`, CI run `45594545137`, and the BRD v1.2 stale `v1.1` footer while preserving original text and PDF SHA-256 hashes.
- **`UNKNOWN` State (`AUD-S2-P3-02`):** `services/site-agent/deployment.go:29-31,287-292` persists `job_stage.state='UNKNOWN'` directly and runs an idempotent corroborated repair on `OpenConnected` without altering `execution_receipt.signed_receipt`. Independently verified via Python `sqlite3` probe plus `TestUnknownStageMigration`.
- **Multi-Site Enrolment (`AUD-S2-P3-03`):** `tests/sprint2/central.go` replaces single-site constants with a durable `enrolment(tenant, site, installation, agent, receipt_key)` table, unique `(installation, sequence)` index, scoped `/next` dispatch, and enrolment-joined `accept()`. Verified in `TestMultiSiteEnrolment` and `TestCentralReceiptBinding`.

---

## 5. Accepted vs. Independently Executed Evidence

| Evidence Category | Independently Executed in Audit Sandbox | Accepted via Hosted CI & Checksummed Local Register |
|---|---|---|
| Candidate source fingerprint (`3efaa1b6...`) | **Yes** (`scripts/verify-sprint2r-evidence.py`) | Matches `source.log` & `index.json` |
| 42 evidence file checksums | **Yes** (`scripts/verify-sprint2r-evidence.py`, 42/42 match) | Registered in `sprint-2r-evidence/index.json` |
| Original audit report SHA-256 (`b0e5d99b...`) | **Yes** (`sha256sum`) | Matches `O1-Sprint-2R-Remediation-Traceability.json` |
| Contract binding drift (`types.ts`, `types_generated.go`) | **Yes** (`node scripts/generate-types.mjs`, 0 diff) | Matches `binding-comparison.log` & hosted CI |
| `UNKNOWN` stage SQLite migration idempotency & receipt immutability | **Yes** (Python `sqlite3` probe) | Matches `TestUnknownStageMigration` in `linux-sprint2.log` |
| Go ordinary & `-race` suites (Windows 86 + Linux 228 + 1 skip) | No (no Go compiler in sandbox) | **Accepted:** Hosted CI runs `45594545137`, `45768586579`, `45768899686` + verified raw logs (`go-windows.log`, `race-windows.log`, `linux-*.log`, `linux-race-*.log`, `red-linux.log`) |
| TypeScript (31/31), Crypto Interop (2/2), PG16 RLS (5/5, 120 tx) | No (no `node_modules`/PG16 in sandbox) | **Accepted:** Hosted CI runs + verified raw logs (`node-test.log`, `node-test-interop.log`, `postgres.log`) |

---

## 6. New Findings Check

**Zero new findings** (`0 P0`, `0 P1`, `0 P2`, `0 P3`) were identified in the Sprint 2R diff. No unrelated historical O0 findings were reopened.

---

## 7. Remaining Production Governance Blockers (Unchanged)

All six production governance blockers remain **OPEN** and strictly outside test-only engineering acceptance:
1. **`AUD-S2-GOV-01`:** `OD01–OD08` ADRs (`0/8` signed) and `Q01–Q07` policy/vendor selections remain unsigned.
2. **`AUD-S2-GOV-02`:** Joint Operations + ZypoSono Clinical/Security `LOCAL-01` sign-off (`ZypoSono_One_Master_BRD_v1.5.pdf` §22.2) remains open.
3. **`AUD-S2-GOV-03`:** OS service-account isolation (separate Linux users/groups, systemd hardening, compromised-agent containment) remains unimplemented (simulation shares UID/GID `65534`).
4. **`AUD-S2-GOV-04`:** Production PKI/CA/KMS, TPM 2.0 key custody, TUF anti-rollback/freeze metadata, and outbound mTLS remain gated.
5. **`AUD-S2-GOV-05`:** Production operator IAM, OIDC IdP, phishing-resistant MFA, and authenticated Ops API ingress remain gated.
6. **`AUD-S2-GOV-06`:** Physical power-loss durability, real OCI/migration execution, authorised hold-release/compensation ceremony, and independent clinical commissioning remain gated.

---

## 8. Final Mandatory Acceptance Verdict Block

```text
AUDITED_COMMIT: 92b3b4059dc96ea4ec78222d6774f4ad16873c23
SOURCE_FINGERPRINT_VERIFIED: YES_3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6
EVIDENCE_INTEGRITY: PASS_42_OF_42_FILES_VERIFIED
HOSTED_CI: PASS_45594545137_45768586579_45768899686
AUD_S2_P1_01: CLOSED_EVIDENCE_ACCEPTED
AUD_S2_P1_02: CLOSED_EVIDENCE_ACCEPTED
AUD_S2_P2_01: CLOSED_EVIDENCE_ACCEPTED
AUD_S2_P2_02: CLOSED_EVIDENCE_ACCEPTED
AUD_S2_P2_03: CLOSED_EVIDENCE_ACCEPTED
AUD_S2_P3_01: CLOSED_INDEPENDENTLY_VERIFIED
AUD_S2_P3_02: CLOSED_INDEPENDENTLY_VERIFIED
AUD_S2_P3_03: CLOSED_EVIDENCE_ACCEPTED
ORIGINAL_FINDINGS_CLOSED: 8_OF_8
ORIGINAL_FINDINGS_OPEN: 0
NEW_P0_FINDINGS: 0
NEW_P1_FINDINGS: 0
NEW_P2_FINDINGS: 0
NEW_P3_FINDINGS: 0
REVOKED_KEY_AUTHORITY_BYPASS: NONE_VERIFIED
CRASH_RECOVERY_VERIFIED: YES_21_MATRIX_CASES_PLUS_CR04_ACCEPTED
DUPLICATE_EXECUTION_PREVENTION: PASS
CONCURRENCY_SAFETY: PASS_BOUNDED_OVERLAPPING_AND_PER_JOB_SERIALIZED
RUNTIME_EVIDENCE_ACCEPTED: YES_HOSTED_CI_AND_42_CHECKSUMMED_LOGS
OVERALL_ENGINEERING_SCORE: 9.2/10
O1_SPRINT_2R_FINAL_VERDICT: GO_TEST_ONLY
GOVERNANCE_BLOCKERS: 6_OPEN (AUD-S2-GOV-01..06)
OD01_TO_OD08_APPROVED: 0_OF_8
O0_PRODUCTION_GATE: NO_GO
PRODUCTION_TRUST_INTEGRATION_ALLOWED: NO
O1_RUNTIME_VERIFIED: NO_FORMAL_GATE_PASS
CLINICAL_PRODUCTION_READY: NO_GO
READY_FOR_O1_SPRINT_3_TEST_ONLY: YES_TEST_ONLY
```
