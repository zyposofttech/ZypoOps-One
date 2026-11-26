# O1 Sprint 2R runtime evidence

Final register prepared 26 November 2026. Baseline `8f9d88be2013c9b014202fc2435ec1368d83575c`, uncommitted candidate source SHA-256 `3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6`. All 42 raw logs have byte-level SHA-256 checksums. The register holds argument arrays, working directory, exit codes, available durations and build environment overrides. Go logs retain named outcomes. Some successful tools produced empty stdout. This is unsigned local evidence. [Machine-readable register](sprint-2r-evidence/index.json)

## Results on the candidate

| Run | Passed | Failed | Skipped | Evidence |
|---|---:|---:|---:|---|
| Windows Go, ordinary | 18 parents + 68 subtests | 0 | 0 | [Go JSON](sprint-2r-evidence/go-windows.log) |
| Windows Go, race | 18 + 68 | 0 | 0 | [Go race JSON](sprint-2r-evidence/race-windows.log) |
| Linux Go, ordinary | 36 + 192 | 0 | 1 | [Sprint 2 ordinary](sprint-2r-evidence/linux-sprint2.log), [all five packages](sprint-2r-evidence/index.json) |
| Linux Go, race | 36 + 192 | 0 | 1 | [Sprint 2 race](sprint-2r-evidence/linux-race-sprint2.log), [all five packages](sprint-2r-evidence/index.json) |
| TypeScript | 31 | 0 | 0 | [Named tests](sprint-2r-evidence/node-test.log) |
| ES256 and Ed25519 TypeScript → Go → TypeScript | 2 | 0 | 0 | [Interop](sprint-2r-evidence/node-test-interop.log) |
| PostgreSQL 16.13, migrations and pooled RLS | 5 | 0 | 0 | [Database](sprint-2r-evidence/postgres.log) |

Each race run repeats its platform coverage. On Linux, `TestServiceProcess` skips when invoked without a child role and executes as the real Agent, Helper or product subprocess in the parent tests. Packages without test files have no test events. Linux package totals in each run were cryptography 29, Helper 19, local policy 27, Agent 13 and Sprint 2 140 passes. All requested local suite categories ran. The candidate hosted CI run remains outstanding.

The unprivileged WSL2 run used kernel 6.6.87.2-microsoft-standard-WSL2 on x86_64, UID/GID 65534. Windows reported 10.0.26200.9974. Go was 1.27.4, Node 24.14.0, pnpm 10.32.1, Prisma Client 6.19.3 and PostgreSQL 16.13. Linux ordinary binaries were cross-compiled with CGO disabled. Linux race binaries were built with CGO, Zig 0.14.1 and the workspace glibc loader, then executed inside Linux. Windows race used clang from llvm-mingw. Compiler, loader and binary paths appear in the register. [Build commands](sprint-2r-evidence/index.json) · [Kernel](sprint-2r-evidence/linux-platform.log) · [Identity](sprint-2r-evidence/linux-identity.log)

## Runtime witnesses

| Area | Witness and observed result |
|---|---|
| Defect before remediation | The reserved and committed cases each failed at Agent restart with `UNTRUSTED_KEY` on the audited implementation. The parent and its two failing subtests yield three failure events and exit 1. This is the only expected nonzero result in the register. [Baseline subprocess log](sprint-2r-evidence/red-linux.log) |
| Crash, replay and redelivery | 21 `TestCrashMatrix` subcases plus CR04 in `TestCommittedDisconnect`. The matrix includes 13 CR09 variants. Original execution ID and consumed grants survive restart. Committed effects remain one. Fenced operations retain zero effects. Unavailable observation and revoked receipt identity retain the hold. [Linux ordinary](sprint-2r-evidence/linux-sprint2.log) · [Linux race](sprint-2r-evidence/linux-race-sprint2.log) |
| Agent and Helper | `TestFaultRecovery100`, `TestExternalKillRecovery`, `TestRebootAndAuthenticatedClockRecovery`, `TestContainment` and `TestKernelSocketPeer` passed. [Agent](sprint-2r-evidence/linux-site-agent.log) · [Helper](sprint-2r-evidence/linux-local-helper.log) |
| Bounded concurrency | `TestBoundedStalledConnections` and `TestOverlappingAdmissions` cover partial frames, saturation, two product barriers, shutdown, duplicate requests and reopen. [Race](sprint-2r-evidence/linux-race-sprint2.log) |
| Preflight and effect | Eight `TestPreflightTransactions` role/fault cases and 14 `TestEffectBoundaries` subtests across two products. Independent ledger writers progressed during preflight. Both roles latched clock rollback. [Barrier and durable-state assertions](sprint-2r-evidence/linux-sprint2.log) |
| Signed backup | 13 cases per preflight mode, including a valid control in each. The two-product `TestAdmissionFailures` harness also passed 54 named adverse subcases. [Policy](sprint-2r-evidence/linux-local-policy.log) · [Integrations](sprint-2r-evidence/linux-sprint2.log) |
| UNKNOWN and enrolment | `TestUnknownStageMigration` compares job, stage, signed receipt, Helper result and central receipt across a legacy-stage repair. `TestMultiSiteEnrolment` exercises receipt identities for two installations, conflict rejection and reopen. [Sprint 2](sprint-2r-evidence/linux-sprint2.log) |
| PostgreSQL isolation | Both migrations applied on a clean temporary cluster. A second migration pass found zero pending. The four-connection pool ran 120 concurrent NestJS/Prisma transactions. All five groups passed, including cross-site/tenant denial and atomic single-use consumption. [Database log](sprint-2r-evidence/postgres.log) |
| Archived O0 pack | 15/15 schemas and 9/9 typed commands, 5/5 LOCAL-01 definitions with PHI-field rejection, SQLite DDL/durability constraints, and 15 signed vectors plus two adversarial negatives in each of Node and Go. [Schema](sprint-2r-evidence/validate_pack.log) · [LOCAL-01](sprint-2r-evidence/local_api_contract_tests.log) · [SQLite](sprint-2r-evidence/sqlite_agent_tests.log) · [Node](sprint-2r-evidence/archived-node.log) · [Go](sprint-2r-evidence/archived-go.log) |
| Static verification | ESLint, TypeScript checking, Go formatting, Go vet for Windows and Linux, and Git diff checks exited 0. Scratch generation reproduced both tracked bindings after line-ending normalisation. [Binding digests](sprint-2r-evidence/binding-comparison.log) · [Commands and exit codes](sprint-2r-evidence/index.json) |

## Reproduction

From the candidate working tree with the repository-pinned toolchains, first run `python scripts/verify-sprint2r-evidence.py`. It hashes the sorted Git-visible runtime paths using LF-normalized content, excluding documentation, the README and .gitignore, then checks the raw bytes of every registered log. The immutable original audit SHA-256 is `b0e5d99bc3c6d7d26b01065e2be2d192795ba4727bbb36be321352a986eb2df6`. The archived O0 ZIP SHA-256 is `580992b65d239f73e1392031251c3415d94ab343f7d4d99aeef322030dbe1f1b`. These identify the documentation inputs separately from the source fingerprint.

The exact Windows and WSL commands executed here are in the register. On a native Linux host use a non-root user with a writable home, temporary directory, CGO-capable toolchain and usable PostgreSQL 16 binaries. The equivalent repository commands are:

```sh
pnpm install --frozen-lockfile
pnpm db:generate
pnpm lint
pnpm typecheck
pnpm test
pnpm test:interop
go vet ./...
go test -count=1 -v ./...
go test -race -count=1 -v ./...
PGBIN=/usr/lib/postgresql/16/bin node infrastructure/local-development/tenancy.mjs
```

For focused review use the following test selectors on Linux:

```sh
go test -race -count=1 -v ./tests/sprint2 -run 'TestCrashMatrix|TestCommittedDisconnect|TestSubstitutedHelperExecution'
go test -race -count=1 -v ./tests/sprint2 -run 'Test(BoundedStalledConnections|OverlappingAdmissions|PreflightTransactions|EffectBoundaries|UnknownStageMigration|MultiSiteEnrolment)$'
go test -count=1 -v ./packages/local-policy -run TestBackupAdmissionModes
```

The original red run added two test-table rows at the audited commit, using the existing applied and reserved barriers and revoking the fixture `mock-job` key on restart. The second competing job must use a valid signer to test its site hold. The baseline log records the actual first-run failures. The original local executable can be invoked with the command in the register. The candidate's expanded table contains both cases.

For drift verification, copy `packages/management-contracts/schemas` and `catalog.json` to the same relative paths under a scratch root. Run the repository's `scripts/generate-types.mjs` with that root as the working directory and compare both generated targets with the tracked files. For archived checks, extract the O0 ZIP so the pack root contains `tests`, `schemas` and `vectors`. Use Python with jsonschema and cryptography for the three archived Python test scripts. Execute the Go and Node vector commands using the paths in the register.

## Hosted evidence and limits

Run 45594545137 reports a successful foundation job on the audited SHA. The supplied hosted baseline counts were 31 TypeScript tests, two interop cases, five PostgreSQL groups and a 72.576-second Linux Sprint 2 package. The retained GitHub response verifies the SHA and job steps. Candidate outputs above come from local executions. The updated workflow can retain named Go outcomes, commit and source fingerprint for 14 days on a future authorised run. [Hosted baseline](https://github.com/zyposofttech/ZypoOps-One/actions/runs/45594545137) · [Retrieved metadata](sprint-2r-evidence/hosted-baseline.log)

Independent acceptance, separate OS service accounts, production trust, physical power-loss and clinical commissioning remain outstanding. The [traceability register](O1-Sprint-2R-Remediation-Traceability.json) maps every technical and governance finding.
