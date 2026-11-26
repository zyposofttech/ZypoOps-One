# O1 Sprint 2 checkpoint

Resumed on main at 135fb6619bc95401263c0b184456cea4615f03de with a clean working tree. GitHub Actions run 45427502336 completed successfully on that SHA. It covers the foundation workflow including Linux integration and PostgreSQL. The existing two-product path, committed-disconnect witness and duplicate-delivery tests are the starting point.

Open work includes process kills at the remaining durable boundaries, expiry and fencing, protection after ambiguous health verification, central identity binding, local interface negatives, and consolidated Windows and Linux evidence. Next step is to exercise the existing state transitions and correct the gaps at their owners. OD01–OD08 await signatures. Independent production NO_GO applies.

## Crash-boundary milestone

The first unprivileged WSL run of TestCrashMatrix passed 11/11 subcases. It killed actual children at pending, reservation, application commit, before receipt delivery and before acknowledgement barriers. It covered product-only restart, both management startup orders, two disconnected recovery attempts, and committed versus fenced outcomes after expiry. Original execution 3f23f691-9aff-5c1b-83a2-64a2ce959097 had one final logical receipt per isolated database. Agent and Helper each consumed one approval. The existing CR-04 integrated test is preserved for the consolidated run.

Helper admission now holds UNKNOWN and post-effect FAILED rows. Central checks deployment state combinations and the enrolled receipt signer. Next step is the conflict and socket negative corpus, followed by one consolidated regression.

## Trust-boundary and maintainability milestone

Both product profiles passed the targeted admission and effect tests. The independent Agent and Helper rejected signed conflicts, reused nonces, sequences and approvals. Product mutation was fenced at backup, grant and job expiry. Linux framing and peer rejection passed. Central rejected invalid first receipts and preserved its signed outcome on reopen. The agent now pins the expected execution before dispatch. A trusted legacy helper reservation can be provisioned for an outstanding pre-upgrade central command. Sprint 2 Go sources are gofmt clean. The final consolidated regression is next, including actual Linux race-instrumented binaries.

## Consolidated regression and handoff

Windows regular and race builds each passed 58 named Go tests/subtests. WSL2 regular and race builds each passed 179 with one standalone role-entry skip. All four Linux test packages ran under UID/GID 65534. The final targeted cross-table assertions passed all 12 crash subcases again in both Linux builds. TypeScript passed 31/31, interop 2/2 and PostgreSQL 16.13 passed 5/5, including 120 transactions on four connections. Lint, type checking, both vet targets, formatting, drift, diff validation and five source-pack harnesses passed.

The complete evidence, scoped limitations, exact commands and verdict are in [the Sprint 2 implementation report](O1-Sprint-2.md). The next step is independent review. The local hardening has no commit or hosted result. OD01–OD08 remain unsigned and Sprint 3 is gated on review.

## Dated status correction · 25 November 2026

The previously local Sprint 2 candidate was subsequently committed as `8f9d88be2013c9b014202fc2435ec1368d83575c`. Hosted run 45594545137 passed on that SHA. The earlier entries above record the state when the original regression ran. The independent audit returned GO_WITH_CONDITIONS_TEST_ONLY. Sprint 3 awaits independent remediation acceptance. [Hosted run](https://github.com/zyposofttech/ZypoOps-One/actions/runs/45594545137)

AUD-S2-P3-01 document erratum. The Operations Platform Master BRD v1.2 carries stale v1.1 footer text. Use the title, revision history and recorded source checksum to identify the revision. The original PDFs and their checksums are preserved.
