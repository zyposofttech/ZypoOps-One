# O1 Sprint 1 engineering foundation

**24 November 2026 · Complete for the test-only Sprint 1 scope.** All four engineering waves have local execution evidence. The PostgreSQL rerun reproduced 120 isolated transactions. Linux kernel credential checks and abrupt child-process termination ran on WSL2. Formal O1 runtime acceptance and clinical production approval remain outstanding.

## Resume point and change record

The worktree was clean at resume on `main`, commit `85e0771162257f355c877ce0b7920d1b90b5002a` (Create Architecture And CodeBase), matching the local `origin/main` reference. The existing implementation already passed the original 15-message corpus, both directions of cross-runtime signing, the database tests, basic SQLite restart tests and Windows helper policy tests. Its clock confidence could become permanently untrusted after a boot change. Kernel socket execution and authenticated recovery had no recorded passing run. The new changes are in the working tree. No commit, push or deployment was performed in the resumed work.

| Wave | Implemented components | Paths |
|---|---|---|
| 1A | Draft 2020-12 validators, schema-derived Go and TypeScript types, strict wire decoding, JCS, P-256/P1363 and Ed25519 signatures, purpose and site binding, revocation and independently signed nested attestations | `packages/management-contracts/`, `packages/signing-ts/`, `packages/signing-go/`, `scripts/generate-types.mjs`, `tests/contract/`, `tests/cryptography/`, `tests/fixtures/` |
| 1B | Two PostgreSQL migrations, 33 introspected Prisma models, non-owner runtime role and transaction-scoped tenant/site context through a Nest application context | `packages/database/`, `services/ops-api/app.ts`, `tests/tenancy/isolation.test.ts`, `infrastructure/local-development/tenancy.mjs` |
| 1C | Installation-bound SQLite WAL ledger, mock outbound transport, replay floor, transactional simulated effect and signed receipt, restart delivery and authenticated clock recovery | `services/site-agent/`, `packages/local-store/sqlite.go`, `packages/local-policy/policy.go` |
| 1D | Separate test trust configuration, independently verified MC-03/05/09, backup proof where required, local preflight, helper-owned replay/consumption database and private Unix socket | `services/local-helper/helper.go`, `socket_linux.go`, `helper_test.go`, `socket_linux_test.go` in the same directory |

**Changes since the resume commit**

- `services/site-agent/agent.go` preserves the UTC watermark across job rejection while rolling back every job-side write. An untrusted initial sample cannot establish a future floor. A changed boot ID invalidates time confidence.
- `services/site-agent/recovery.go` adds a locally configured loopback TLS test adapter. It checks the exact peer certificate, boot, monotonic reading, persisted UTC floor and freshness before restoring confidence. The two-second agreement bound is a fixture policy pending Q03. Production time recovery remains a separate decision.
- `services/site-agent/recovery_test.go`, `idempotency_test.go` and `kill_test.go` exercise reboot, rollback, wrong and stale time proofs, distinct-job idempotency conflicts, and external process termination on each side of commit.
- `services/local-helper/helper.go` preserves the clock floor on rejection. `helper_test.go` checks fresh signed jobs with consumed grants, rollback of partial consumption, independently valid wrong-purpose signatures and tampered jobs. `socket_linux_test.go` exercises matching credentials, wrong UID and wrong GID on a real Unix socket.
- `packages/signing-ts/index.ts` validates an approval candidate before issuing a nested signature. `tests/contract/contracts.test.ts` covers invalid and changed target digests. `tests/cryptography/rejection_test.go` independently exercises Go verifier and schema rejection paths.
- `infrastructure/local-development/tenancy.mjs` keeps the disposable Linux PostgreSQL cluster on loopback TCP. `.github/workflows/foundation.yml` runs generation drift checks, TypeScript, Go race tests, Linux sockets and PG16 integration. Hosted CI execution remains pending a user-authorised push.
- `docs/implementation/source-sha256.txt`, `finding-lineage.json` and the three corrected reading copies under `specifications/` record source provenance and O0 hygiene changes. This file is the consolidated report.

## Execution evidence

Versions used were Node 24.14.0, pnpm 10.32.1, TypeScript 5.9.3, Go 1.27.4, PostgreSQL 16.13, Prisma 6.19.3 and modernc SQLite driver 1.44.3. Exact transitive dependencies are in `pnpm-lock.yaml` and `go.sum`. Commands below ran in the repository root unless noted. Windows used the workspace-local Go toolchain and module/build caches under `.work/`.

| Executed command | Observed result |
|---|---|
| `pnpm.cmd generate` followed by `git diff --exit-code -- packages/management-contracts` | Generated files reproduced with zero diff |
| `pnpm.cmd lint` and `pnpm.cmd typecheck` | Both exited 0 |
| `pnpm.cmd test` | 31 passed, 0 failed. Includes all 15 signed fixtures, nine discriminated commands, JCS input limits, signed consent and negative cases |
| `pnpm.cmd test:interop` | 2 passed, 0 failed. Fresh TS signature verified in Go and fresh Go signature verified in TS, separately for ES256 and Ed25519 |
| `go vet ./...` | Exit 0 |
| `go test -json -count=1 ./...` and `go test -count=1 ./...` | Windows, 15 top-level tests, 42 subtests, zero failures. Of the 57 total pass events, 29 were crypto, 13 agent and 15 helper |
| `go test -race -count=1 ./...` | Windows, all three packages with tests passed. Enabled CGo with workspace-local LLVM/MinGW 20261117 UCRT and `CC=clang`. The initial attempt with CGo disabled could not start |
| `node infrastructure/local-development/tenancy.mjs` with `PGBIN=C:\Program Files\PostgreSQL\16\bin` | 5 groups passed, 0 failed, 0 skipped. Both migrations applied. Repeat deployment reported `No pending migrations to apply.` The runner stopped the disposable server |
| Linux commands below | Helper 3 top-level tests and 16 subtests passed. Agent 8 top-level tests and 5 subtests passed. Zero failures. These executions overlap the Windows coverage |
| `git diff --check` | Exit 0 |

Linux test commands used the installed WSL2 distribution and unprivileged user. The three `SO_PEERCRED` cases read kernel-supplied credentials on connected sockets. The Linux agent binary also executed its own subprocess crash witnesses.

```powershell
$env:GOOS='linux'
$env:GOARCH='amd64'
$env:CGO_ENABLED='0'
go test -c ./services/local-helper -o .work/helper-linux.test
go test -c ./services/site-agent -o .work/agent-linux.test
wsl.exe -d docker-desktop -u nobody --exec /mnt/host/d/ZypoOps-One/.work/helper-linux.test '-test.v'
wsl.exe -d docker-desktop -u nobody --exec /mnt/host/d/ZypoOps-One/.work/agent-linux.test '-test.v'
```

The source-pack regression commands were also executed. The Python executable was the bundled `codex-primary-runtime/dependencies/python/python.exe` with workspace-local `PYTHONPATH=.work/pythonlibs`. The source pack is the verification ZIP nested in the complete remediation ZIP.

| Source-pack command | Result |
|---|---|
| `python .work/pack/tests/validate_pack.py` | 15/15 schema fixtures, 9/9 command shapes and synthetic enrolment structural checks passed |
| `python .work/pack/tests/local_api_contract_tests.py` | 5/5 LOCAL-01 payload shapes and forbidden field rejection passed |
| `python .work/pack/tests/sqlite_agent_tests.py` | One structural invariant suite passed |
| `node .work/pack/tests/verify_vectors.mjs` | 15/15 signed fixtures, two adversarial vectors and strict-parser checks passed |
| `go run .work/pack/tests/verify_vectors.go .work/pack` | 15/15 historical ASCII fixtures, two adversarial vectors and duplicate rejection passed. The active Go verifier uses the separate RFC 8785 implementation |

## Security evidence and coverage boundaries

**Contracts.** The wire version is exactly `1.2.0` for MC-01 through MC-14 and AUX-01. LOCAL-01 uses the five definitions in its `v1.2.0` schema. The supplied MC schema resource IDs retain their historical `v1.1.0` suffix. Dispatch checks the envelope version separately. MC-05/06/10/12 are globally artifact-scoped. The remaining envelopes require site scope. Each runtime reproduced the original signing bytes and passed a shared Unicode/number corpus. A valid new central signature could not transfer an MC-09 approval across sites. The signature tests cover software keys. Hardware custody and independent implementation certification remain open.

**Database.** Migration 001 matches the supplied SQL byte-for-byte. Migration 002 adds customer legal entities, global and site-specific trust records, site-scoped policies, the rollout artifact FK, constrained job transitions, guarded approval consumption, restricted grants and composite uniqueness for Prisma relations. The test cluster used a separate migration login and a non-owner, non-superuser, NOBYPASSRLS runtime login. All 31 scoped tables had ENABLE and FORCE RLS. The two global data tables were separately scoped. Queries without context returned zero sites and writes were rejected. The tests rejected foreign site ownership, cross-tenant and sibling-site access, duplicate idempotency keys and self-approval. Concurrent consumption produced one winner. All 120 transactions wrote a health row and read only their own site through a four-connection pool. The application must supply authenticated context to the wrapper. Production IAM remains gated.

**Agent.** On Windows and Linux, the 100-iteration fault loop failed inside the SQLite transaction, reopened the database and recovered exactly 100 simulated effects. Separate tests terminated a child inside the transaction and killed children before and after commit. Recovery observed zero pending receipts before commit and one after commit. Redelivery produced one effect, one signed terminal receipt and two consumed grants. A simulated local application continued serving during agent death. Offline receipt delivery resumed after reopening the ledger. Key rotation preserved the replay floor. Out-of-order jobs and different jobs reusing an idempotency key failed without advancing sequence. Boot changes and wall/monotonic regressions persisted UNTRUSTED confidence until authenticated recovery. The reboot case changes the host-clock inputs and reopens the real database. Physical power loss, hardware clocks and an actual clinic-host reboot were outside this run.

**Helper.** The helper read UID and GID from the Linux kernel and rejected callers that failed either check. An approved test deployment produces a SQLite simulation record. The only accepted operation is a signed, allow-listed test deployment with the required artifact, approval and preflight evidence. Caller-controlled mounts and executable paths have no execution implementation. The suite rejected modified or unsigned jobs/manifests, wrong-purpose signatures, wrong sites, missing or expired authority, unapproved artifacts, locked products, replayed jobs and reused grants. Rejection of a mixed fresh/consumed set left its fresh grant available. Restart preserved the replay ledger. The helper and agent were exercised as separate components. A production systemd configuration, OS ownership separation between two installed service accounts, TUF, Podman and real deployment effects remain unverified.

**Traceability.** The implemented tests exercise portions of AT06, AT09, AT47, NX05, NX07, NX13, NX44 and NX58. Contract cases include CT-O0-01/03/04/07/08/15/16/17/18. Hardware-dependent, full customer workflow and full compromise scenarios retain separate acceptance gates. Local runs do not confer blanket acceptance on the 50 AT or 70 NX entries.

## O0 hygiene and authority

All seven required inputs were located and read. The two SQLite/PostgreSQL baseline SQL files match their archive members. The fingerprint file identifies the actual standalone PDFs and the nested contract pack. The repository's existing ignore rules keep the ZIP and CSV local. The direct Ops BRD and consolidated specification PDFs differ bytewise from their archive copies. The corrected reading copies derive from the standalone PDFs; the implementation derives from the nested contract pack.

MC-10 now limits the nullable approval reference to 128 characters. LOCAL-01 rejects unknown root fields and continues validating each endpoint definition independently. The revised reading copies correct active document labels, the SQLite filename, the 15 signed-message count and the 16-schema inventory including LOCAL-01. Historical revision entries are retained. All 109 pages of the revised documents were rendered and representative changed regions visually checked. The independent audit's historical Go execution status is distinguished from this sprint's actual run.

The lineage file preserves the 27 original identifiers and titles from the pack alongside the independent ledger's same-ID descriptions. Topic links are advisory. For example, the original AUD-P0-07 describes helper isolation, while the newer ledger discusses helper isolation at AUD-P0-05. The independent audit's closure decisions remain authoritative. No historical finding was reopened or independently closed here.

## Governance register

Every decision below is **proposed and unsigned**. The accountable roles and approval evidence are in `docs/adr/OD01_FOR_APPROVAL.md` through `OD08_FOR_APPROVAL.md`. Each requires an identified human owner, review of the source and threat implications, resolution of the listed dependency, and signed acceptance. Joint Operations/ZypoSono engineering and clinical approval of the local interface also remains outstanding.

| Decision | Accountable roles | Decision and evidence required to unblock production |
|---|---|---|
| OD01 | Product Sponsor, Architecture Authority, Clinical Product Owner | Q06. Approve authority boundaries, local endpoint ownership and fail-closed maintenance locks |
| OD02 | Security Authority, Engineering Owner | Q02. Approve TPM availability, protected identity, fallback hosts, key custody and replacement |
| OD03 | Security Authority, Engineering Owner | Q01. Select and approve CA/KMS, trust-root rotation, certificate policy, revocation and expiry recovery |
| OD04 | Security Authority, Product Sponsor | Q04. Select supported TUF metadata and custody, approve signing and site-consent ceremonies and helper trust |
| OD05 | Security Authority, Architecture Authority | Q05. Approve IdP, account lifecycle, break-glass and runtime tenancy controls |
| OD06 | Architecture Authority, Product Sponsor | Approve OCI/rootless execution, product interface and rollout scope. The source ADR gives no numbered Q association; Q04 and Q06 are related integration dependencies |
| OD07 | Privacy/Legal Owner, Clinical Owner | Q07. Approve technical data classes, retention and any exception basis |
| OD08 | Product Sponsor, Privacy/Legal Owner, Clinical Owner | Q03. Approve grace, clock recovery and non-destructive post-grace policy |

## Next engineering wave and verdict

Run a two-product, unprivileged end-to-end simulation through the actual agent-to-helper socket. Verify product-specific LOCAL-01 locks/readiness, signed backup evidence, durable helper-to-agent receipt reconciliation after a disconnect, and rollback on failed health checks. Use test keys and mock effects. Production trust integration, deployment privileges, protocol freeze and clinical release require their separate approvals. No failing requested test or unresolved defect in the exercised Sprint 1 paths is known.

```text
O1_SPRINT_1_FOUNDATION_STATUS: COMPLETE_TEST_ONLY
WAVE_1A_CONTRACTS_AND_CRYPTO: PASS_TEST_CORPUS
WAVE_1B_DATABASE_AND_RLS: PASS_POSTGRESQL_16_13
WAVE_1C_SITE_AGENT: PASS_TEST_ONLY
WAVE_1D_LOCAL_HELPER_PROTOTYPE: PASS_UNPRIVILEGED_LINUX_PROTOTYPE
NODE_TYPESCRIPT_TESTS: PASS
GO_TESTS: PASS
POSTGRESQL_INTEGRATION_TESTS: PASS
SQLITE_DURABILITY_TESTS: PASS_WITH_PROCESS_FAULT_INJECTION
SECURITY_NEGATIVE_TESTS: PASS_TESTED_CASES
CROSS_RUNTIME_CRYPTO_TESTS: PASS_BOTH_DIRECTIONS_BOTH_ALGORITHMS
O0_P2_HYGIENE_CORRECTIONS: IMPLEMENTED_AWAITING_INDEPENDENT_CLOSURE
OD01_TO_OD08_APPROVED: 0_OF_8
GOVERNANCE_BLOCKERS: OD01_TO_OD08; Q01_TO_Q07; JOINT_LOCAL_INTERFACE_SIGN_OFF
PRODUCTION_TRUST_INTEGRATION_ALLOWED: NO
PRIVILEGED_CUSTOMER_DEPLOYMENT_ALLOWED: NO
O1_RUNTIME_VERIFIED: NO_FORMAL_GATE_PASS
CLINICAL_PRODUCTION_READY: NO_GO
READY_FOR_NEXT_ENGINEERING_WAVE: YES_TEST_ONLY
```
