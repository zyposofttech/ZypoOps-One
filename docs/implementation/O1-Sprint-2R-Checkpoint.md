# O1 Sprint 2R checkpoint

Clean main, no staged or untracked files, baseline 8f9d88be2013c9b014202fc2435ec1368d83575c. Read all 18 sections of the independent audit in Downloads/o1_sprint_2_independent_audit.md. Original SHA-256 b0e5d99bc3c6d7d26b01065e2be2d192795ba4727bbb36be321352a986eb2df6. Engineering verdict GO_WITH_CONDITIONS_TEST_ONLY. Eight technical findings and six governance blockers. Next action is the red Linux process-restart witness for AUD-S2-P1-01, followed by R1–R6.

## R1 · AUD-S2-P1-01

The baseline reserved and committed Linux subprocess witnesses both failed with UNTRUSTED_KEY. After the local-queue and helper-reservation recovery paths were separated, all 12 CR09 subcases passed. Matched immutable bundle and grant evidence permits observation or fencing. An agent-only admission or unavailable receipt signer stays pending. Changed agent.go, both deployment.go files, and the workflow/crash harness. The baseline CI run 45594545137 was confirmed on the audited SHA.

## R2–R4 implementation

The helper socket now bounds active workers and serializes matching job IDs. Both ledgers release write transactions before LOCAL-01 I/O. Admission rechecks deadlines and site holds. Backup policy now applies to both preflight modes. Windows Agent/Helper regressions passed. Changed socket_linux.go, helper.go, deployment.go, agent.go, policy.go and site.go. Next action is deterministic overlap, stalled preflight, expiry and commit-failure verification.

## R2–R5 targeted evidence

Linux TestOverlappingAdmissions observed both product preflights concurrently, then held the first mutation while a competing job retried with zero grant consumption. TestBoundedStalledConnections verified two worker slots, a partial frame, another completed lookup, rejected saturation and shutdown. Its first test setup briefly occupied a slot with a readiness connection. Waiting for socket creation corrected the fixture. Six SQLite preflight cases verified independent writes, cancellation, expiry and rollback after insertion failure for both ledgers. UNKNOWN migration, 26 backup-policy cases, and persisted two-site enrolment passed targeted runs. Changed tests/sprint2, tests/fixtures/site.go, packages/local-policy, the central simulator, README and dated historical errata. Next action is the consolidated regression and checksummed evidence register.

## R6 · Consolidated regression and evidence handoff · 26 November 2026

The final clock sample and rollback changes were rebuilt for Windows and Linux. Both preflight roles latched rollback in ordinary and race runs. The Linux race Sprint 2 binary finished in 102.422 seconds under UID/GID 65534. Each Linux run recorded 36 parent and 192 subtest passes and one standalone child entry skip. Each Windows run recorded 18 parent and 68 subtest passes. TypeScript passed 31/31, cross-runtime cryptography 2/2 and PostgreSQL 5/5, including 120 concurrent transactions with four connections. All recorded candidate commands exited zero. The two expected pre-fix failures are retained separately.

The source fingerprint is `3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6` on baseline `8f9d88be2013c9b014202fc2435ec1368d83575c`. The 42-log register includes command arguments, exit status, SHA-256 and Go outcome counts. The [implementation report](O1-Sprint-2R.md), [runtime evidence](../audit/O1-Sprint-2R-Runtime-Evidence.md) and [traceability register](../audit/O1-Sprint-2R-Remediation-Traceability.json) identify source paths, dispositions and compatibility limits.

The candidate is local and uncommitted. The next action is focused independent review of this fingerprint, including replaying the Linux recovery and overlap witnesses and accepting or returning each finding. Hosted candidate CI, independent acceptance and the six governance decisions remain outstanding. Sprint 3 is gated.
