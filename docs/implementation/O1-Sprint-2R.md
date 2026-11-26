# O1 Sprint 2R engineering remediation

26 November 2026 · Synthetic management execution · Local candidate on `main`

**The candidate is ready for focused independent re-audit.** Seven findings have implementation or documentation corrections. AUD-S2-P1-02 concerns reviewer access to runtime evidence. All eight await independent review. Six governance blockers remain open.

## Baseline and candidate

Work began on clean `main` at `8f9d88be2013c9b014202fc2435ec1368d83575c` (`O1-Sprint 2 Completed`). The independent audit scored engineering 8.5/10 with GO_WITH_CONDITIONS_TEST_ONLY, zero P0, two P1, three P2 and three P3 findings. Production and clinical usage received NO_GO. The archived original has SHA-256 `b0e5d99bc3c6d7d26b01065e2be2d192795ba4727bbb36be321352a986eb2df6`. [Original independent audit](../audit/O1-Sprint-2-Independent-Audit.md)

Hosted run 45594545137 succeeded on that exact commit. The remediation candidate is uncommitted with source SHA-256 `3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6`. Candidate verification ran locally. A hosted candidate run is outstanding. [Baseline run](https://github.com/zyposofttech/ZypoOps-One/actions/runs/45594545137) · [Candidate evidence](../audit/O1-Sprint-2R-Runtime-Evidence.md)

## Findings and corrections

### AUD-S2-P1-01 · Revoking the ops_job key wedges pending reconciliation

Both roles required live MC-03 trust before consulting their independent durable admission records. Killing the process after reservation or product commit and revoking the issuer made both restart witnesses fail with UNTRUSTED_KEY. Fresh deliveries require current proofs. The Agent resumes its own durable queue. The Helper binds observation or fencing to its own original reservation and consumed grants. A retired issuer cannot authorise a new effect. Missing evidence or an unavailable signer keeps work pending.

Changed files include `services/site-agent/agent.go`, `services/site-agent/deployment.go`, `services/local-helper/deployment.go`, `tests/sprint2/crash_linux_test.go`, `tests/sprint2/workflow_linux_test.go`. Verification includes `TestCrashMatrix`, `TestCommittedDisconnect`, `TestSubstitutedHelperExecution`. [Recorded evidence](../audit/sprint-2r-evidence/red-linux.log)

### AUD-S2-P1-02 · Runtime evidence unavailable to the original reviewer

The audit sandbox lacked the toolchains and access to the hosted run. The implementation required reproducible evidence attributable to a particular candidate. Archived the successful hosted baseline. The local source register ties Windows, Linux, database and cross-runtime evidence to the working tree. The workflow now retains named Go outcomes and source identity for a future authorised run.

Changed files include `.github/workflows/foundation.yml`, `scripts/verify-sprint2r-evidence.py`. Verification includes `TestCrashMatrix`, `TestKernelSocketPeer`. [Recorded evidence](../audit/sprint-2r-evidence/hosted-baseline.log)

### AUD-S2-P2-01 · Serial Helper handlers conceal contention and block unrelated callers

The accept loop completed an entire connection before accepting another. Queued kernel connections could pass a nominal concurrency test without ever overlapping handlers. An explicit worker bound rejects excess connections before starting a worker. Per-job serialisation, durable site holds, deadlines and cancellation protect concurrent workers. Shutdown drains workers. Independent product barriers prove concurrent entry.

Changed files include `services/local-helper/socket_linux.go`, `services/local-helper/helper.go`, `services/local-helper/deployment.go`, `tests/sprint2/remediation_linux_test.go`. Verification includes `TestOverlappingAdmissions`, `TestBoundedStalledConnections`, `TestConcurrentDuplicates`, `TestCompetingOperations`. [Recorded evidence](../audit/sprint-2r-evidence/linux-sprint2.log)

### AUD-S2-P2-02 · Live preflight holds a SQLite writer

Agent and Helper mixed external preflight with durable grant and execution writes. A slow product held the single SQLite writer. Moving the call also exposed an initial clock sample that cancellation could discard. Each role persists the initial time sample, releases the writer for external preflight and rechecks authority at reservation. Grant consumption and execution admission commit together. The product checks final lock, readiness and expiry at the effect boundary.

Changed files include `packages/local-policy/policy.go`, `services/site-agent/agent.go`, `services/site-agent/deployment.go`, `services/local-helper/helper.go`, `services/local-helper/deployment.go`, `tests/sprint2/remediation_linux_test.go`, `tests/sprint2/failures_linux_test.go`. Verification includes `TestPreflightTransactions`, `TestOverlappingAdmissions`, `TestEffectBoundaries`, `TestRejectedJobPreservesClockWatermark`. [Recorded evidence](../audit/sprint-2r-evidence/linux-sprint2.log)

### AUD-S2-P2-03 · MC-14 backup checks depend on the preflight mode

A LocalCheck condition omitted the independently provisioned backup requirement in callback mode. Its old synthetic fixtures also lacked an operation-specific expected manifest. Backup-required jobs use the same signed MC-14 validation before either preflight implementation. Legacy fixtures independently pin their test jobs. Both product integrations exercise the stricter checks.

Changed files include `packages/local-policy/policy.go`, `packages/local-policy/backup_test.go`, `tests/fixtures/site.go`, `tests/sprint2/failures_linux_test.go`. Verification includes `TestBackupAdmissionModes`, `TestAdmissionFailures`, `TestEffectBoundaries`. [Recorded evidence](../audit/sprint-2r-evidence/linux-local-policy.log)

### AUD-S2-P3-01 · Status and controlled-document versions disagree

Time-specific Sprint 2 status had been read as current status. The Operations Platform Master BRD v1.2 retains stale v1.1 footer text. Updated the current README and appended dated corrections to the historical implementation report and checkpoint. A controlled erratum identifies the BRD by its title, revision and recorded checksum.

Changed files include `README.md`, `docs/implementation/O1-Sprint-2.md`, `docs/implementation/O1-Sprint-2-Checkpoint.md`. [Recorded evidence](../audit/sprint-2r-evidence/hosted-baseline.log)

### AUD-S2-P3-02 · The stage projection misrepresents UNKNOWN as FAILED

A terminal-stage mapping collapsed an unresolved post-verification outcome into the ordinary failure state. Persist UNKNOWN directly. Opening a connected ledger repairs the legacy POST_VERIFY projection only where both the immutable receipt outcome and the job state corroborate UNKNOWN. Compare the signed bytes across reopen and central acceptance.

Changed files include `services/site-agent/deployment.go`, `tests/sprint2/remediation_linux_test.go`. Verification includes `TestUnknownStageMigration`, `TestEffectBoundaries`. [Recorded evidence](../audit/sprint-2r-evidence/linux-sprint2.log)

### AUD-S2-P3-03 · Central receipt verification relies on single-site fixture constants

The simulator selected an installation, Agent and signer from constants. A global sequence uniqueness rule also conflated independent installations. Trusted setup provisions durable site enrolments. Submission, scoped delivery and receipt acceptance bind each job to its enrolled installation, Agent and signing key. A local enqueue ordinal supports equal signed sequences at separate installations.

Changed files include `tests/sprint2/central.go`, `tests/sprint2/central_test.go`, `tests/sprint2/workflow_linux_test.go`. Verification includes `TestMultiSiteEnrolment`, `TestCentralReceiptBinding`, `TestLocalAuthority`. [Recorded evidence](../audit/sprint-2r-evidence/go-windows.log)

## Historical authority and restart outcomes

The canonical MC-03 digest and complete bundle binding commit to the original signed inputs, including the artifact, authorising key and operation. The Agent joins its local identity, verified command, pending bundle and consumed grant hashes. The Helper authenticates the socket peer, checks configured scope and target, and joins its independent job digest, reservation, execution identity and consumed grant IDs. Those records originate from admission transactions. A newly signed assertion of past admission cannot create either record through the recovery interface. The roles share a synthetic OS identity. Host compromise and role-specific file ownership remain AUD-S2-GOV-03.

| Durable state at restart | Permitted recovery and observed state |
|---|---|
| Agent admission only, ops key revoked | Keep RUNNING and record a recovery hold. Zero Helper reservations, product effects and central receipts. Delayed execute is denied. |
| Helper reservation, product reports absent | Fence the original execution at the product and persist its rejection. Zero effects, including a delayed activation. |
| Product effect committed | Look up the original execution and complete post-verification. The effect count stays one across repeated interruptions. |
| Helper already terminal or Agent receipt awaiting acknowledgement | Return the bound result or redeliver the original signed bytes. One central terminal row and one consumption per grant in each admitting ledger. |
| Product observation unavailable | Keep the unresolved reservation and Agent hold. A competing product retries. The original effect is reconciled once observation returns. |
| Receipt-signing identity revoked | Keep the pending Agent job and existing Helper evidence. Central receives no new trusted receipt. |

The primary regression first failed on the baseline for both reserved and committed jobs. The candidate passed 21 crash-matrix subcases plus the separate CR04 committed-disconnect witness. Both ordinary and race runs executed real Linux child-process termination and restart. The Site Agent also passed its separate 100-iteration simulated recovery and external-kill regressions. [Baseline failure](../audit/sprint-2r-evidence/red-linux.log) · [Linux crash matrix](../audit/sprint-2r-evidence/linux-sprint2.log) · [Site Agent tests](../audit/sprint-2r-evidence/linux-site-agent.log)

## Concurrency, preflight and backup checks

The Helper defaults to 16 connections and accepts limits of 1–256. A stalled partial frame coexisted with a successful lookup. With two occupied slots, a third connection was rejected and shutdown drained both workers. Two distinct product preflights reached independent barriers. While the first product held the mutation, the second retried. After concurrent requests and a Helper reopen, the ledgers contained one reservation, one consumed grant and product effect counts of one and zero. [Linux race witnesses](../audit/sprint-2r-evidence/linux-race-sprint2.log)

Each Agent and Helper preflight case held a real LOCAL-01 response behind a barrier. A second SQLite connection acquired a writer with an 80 ms busy timeout while the endpoint remained blocked. Cancellation, grant expiry, database insertion failure and clock rollback each produced zero grant consumption and zero effects in both roles. Both rollback cases checked the durable untrusted-clock state. Existing product-boundary cases cover lock changes and backup expiry before mutation. [Transaction and effect-boundary results](../audit/sprint-2r-evidence/linux-sprint2.log)

Signed MC-14 evidence must satisfy enrolled signer and purpose, tenant/site, operation, product, artifact, expected manifest digest, exact categories, timestamp, signed age, freshness, encryption, integrity and error status. Each preflight mode ran one valid and twelve adverse cases. Separate product integration tests exercised tampered proofs, missing consent, unavailable endpoints and scoped signers. [Policy cases](../audit/sprint-2r-evidence/linux-local-policy.log) · [Product integrations](../audit/sprint-2r-evidence/linux-sprint2.log)

## Compatibility and additional defects

- Connected Agent startup adds a recovery-hold table and repairs corroborated UNKNOWN stage projections. Existing signed receipts retain their bytes. Rows predating a durable expected execution pin can reconcile under current trust. Historical terminalisation under revoked authority requires the pin. Missing evidence remains pending for authorised recovery.
- The central simulator adds a durable enrolment table and installation/sequence index. A pre-upgrade store needs trusted provision of its scoped enrolment on first upgrade. Subsequent opens use the persisted association. Duplicate or conflicting provisions are rejected. Old global sequence values remain queue ordinals, with the original installation sequence retained in the signed envelope.
- A test readiness connection could briefly occupy a saturated worker slot. The fixture now waits for socket creation. Review of cancellation also exposed a lost initial clock watermark and led to the two rollback-latch tests. No candidate regression failure remains in the recorded suites.
- These migrations concern synthetic SQLite stores. The PostgreSQL migration set and MC-03/MC-04 schemas retain their audited versions.

## Verification and handoff

Windows recorded 18 parent and 68 subtest passes per ordinary and race run. Linux recorded 36 parent and 192 subtest passes per ordinary and race run. Linux skipped the standalone child-process entry point once per run and invoked it in subprocesses. TypeScript passed 31/31, interoperability 2/2 and PostgreSQL 5/5. The database harness applied both migrations, reported zero pending on the second run and exercised 120 concurrent NestJS/Prisma transactions through four connections. Lint, type checking, formatting, both vet targets, contract regeneration and Git diff validation passed. [Commands, exit codes and raw logs](../audit/O1-Sprint-2R-Runtime-Evidence.md)

The next engineering wave is a **focused independent re-audit**. Verify the baseline-to-candidate diff and evidence hashes. Reproduce the two red cases and the candidate recovery and fence cases. Review both local authority boundaries and failure holds, repeat the Linux race tests, and check MC-14, UNKNOWN migration and scoped enrolment. Record a reviewer disposition for each finding. A hosted candidate run requires separate authorisation to publish the candidate. Sprint 3 awaits independent acceptance. [Traceability register](../audit/O1-Sprint-2R-Remediation-Traceability.json)

- AUD-S2-GOV-01 — OD01–OD08 Human Sign-Off
- AUD-S2-GOV-02 — Joint LOCAL-01 Approval
- AUD-S2-GOV-03 — OS Service-Account Isolation
- AUD-S2-GOV-04 — Production Trust & Custody
- AUD-S2-GOV-05 — Operator IAM & Central API Ingress
- AUD-S2-GOV-06 — Physical Power-Loss, Real OCI Execution, Hold Release & Clinical Commissioning

OD01–OD08 have zero approvals. Q01–Q07 require accountable human decisions. Formal O1 runtime verification, production trust integration and clinical commissioning remain gated.

## Final verdict

```
O1_SPRINT_2R_STATUS: ENGINEERING_COMPLETE_AWAITING_FOCUSED_INDEPENDENT_REAUDIT
BASELINE_COMMIT: 8f9d88be2013c9b014202fc2435ec1368d83575c
AUD_S2_P1_01: REMEDIATED_AWAITING_INDEPENDENT_VERIFICATION
AUD_S2_P1_02: EVIDENCE_PROVIDED_AWAITING_INDEPENDENT_ACCEPTANCE
AUD_S2_P2_01: REMEDIATED_AWAITING_INDEPENDENT_VERIFICATION
AUD_S2_P2_02: REMEDIATED_AWAITING_INDEPENDENT_VERIFICATION
AUD_S2_P2_03: REMEDIATED_AWAITING_INDEPENDENT_VERIFICATION
AUD_S2_P3_01: REMEDIATED_AWAITING_INDEPENDENT_VERIFICATION
AUD_S2_P3_02: REMEDIATED_AWAITING_INDEPENDENT_VERIFICATION
AUD_S2_P3_03: REMEDIATED_AWAITING_INDEPENDENT_VERIFICATION
REVOKED_OPS_JOB_RECOVERY: PASS_TEST_ONLY
COMMITTED_EFFECT_RECONCILIATION: PASS_ORIGINAL_EXECUTION_ONE_EFFECT
UNAPPLIED_RESERVATION_FENCING: PASS_DURABLE_FENCE_ZERO_EFFECTS
NEW_EFFECT_WITH_REVOKED_KEY_DENIED: PASS
HISTORICAL_ADMISSION_AUTHENTICITY: PASS_SYNTHETIC_LEDGERS_AND_AUTHENTICATED_PEER
BOUNDED_HELPER_CONCURRENCY: PASS
ACTUAL_OVERLAPPING_HANDLER_TESTS: PASS_LINUX_ORDINARY_AND_RACE
SQLITE_TRANSACTION_BOUNDARIES: PASS_BOTH_LEDGERS
MC14_ALL_PATHS_ENFORCED: PASS_LEGACY_AND_CONNECTED
UNKNOWN_STATE_CONSISTENCY: PASS_FIVE_STORES_ORIGINAL_RECEIPT_PRESERVED
MULTI_SITE_TEST_ENROLMENT: PASS_TWO_SITE_RESTART
CRASH_MATRIX_REGRESSION: PASS_21_MATRIX_CASES_PLUS_SEPARATE_CR04
WINDOWS_GO_TESTS: PASS_18_PARENTS_68_SUBTESTS_0_SKIPPED
LINUX_GO_TESTS: PASS_36_PARENTS_192_SUBTESTS_1_CHILD_ENTRY_SKIPPED
GO_RACE_TESTS: PASS_WINDOWS_86_LINUX_228
TYPESCRIPT_TESTS: PASS_31_OF_31
CRYPTO_INTEROP: PASS_2_OF_2
POSTGRESQL_TESTS: PASS_5_OF_5_120_TRANSACTIONS_POOL_4
SECURITY_NEGATIVE_TESTS: PASS_TEST_ONLY
HOSTED_CI_BASELINE: PASS_45594545137
HOSTED_CI_REMEDIATION_CANDIDATE: UNRUN_UNCOMMITTED_CANDIDATE
RUNTIME_EVIDENCE_PACKAGE: 42_UNSIGNED_LOGS_SHA256_VERIFIED
TECHNICAL_FINDINGS_REMEDIATED: 7_IMPLEMENTED_1_EVIDENCE_PROVIDED_0_INDEPENDENTLY_ACCEPTED
GOVERNANCE_BLOCKERS: 6_OPEN
OD01_TO_OD08_APPROVED: 0_OF_8
O0_PRODUCTION_GATE: NO_GO
PRODUCTION_TRUST_INTEGRATION_ALLOWED: NO
O1_RUNTIME_VERIFIED: NO_FORMAL_ACCEPTANCE
CLINICAL_PRODUCTION_READY: NO_GO
READY_FOR_FOCUSED_INDEPENDENT_REAUDIT: YES
READY_FOR_O1_SPRINT_3: NO_PENDING_INDEPENDENT_ACCEPTANCE
```
