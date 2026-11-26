# O1 Sprint 2 implementation and verification

25 November 2026 · Synthetic, unprivileged two-product simulation

## Scope and baseline

Work resumed on a clean `main` at `135fb6619bc95401263c0b184456cea4615f03de` (`O1 Sprint 2`). That commit already contained Waves 2A–2F, including the real socket workflow, SQLite ledgers, product profiles, and committed-disconnect test. The foundation job passed contract drift, TypeScript, interoperability, Go vet, Linux Go race and PostgreSQL integration on that SHA. The latest hosted main run still points to that commit. The hardening changes are local and uncommitted. [Hosted baseline run 45427502336](https://github.com/zyposofttech/ZypoOps-One/actions/runs/45427502336)

The implementation follows the existing v1.2 O0 remediation and Operations BRD, v1.5 ZypoSono BRD, proposed OD01–OD08, and MC-03/04/05/09/14, AUX-01 and LOCAL-01. Contract sources and generated bindings have zero tracked drift. [Sprint 1 baseline and source lineage](O1-Sprint-1.md) · [Specification copies](specifications/) · [Proposed ADRs](../adr/)

## Completed waves

| Wave | Implemented path | Local verification |
|---|---|---|
| 2A · Agent–Helper | `packages/deployment/{protocol.go,client_linux.go}`, `packages/unixpeer/peer_linux.go`, `services/local-helper/socket_linux.go` | Real Unix socket, bilateral UID/GID pinning, bounded framing, cancellation, stale socket recovery |
| 2B · Products | `products/{zyposono-sim,mock-secondary}/profile_linux.go`, `products/simulator/application_linux.go` | Same agent executable and helper served two product identities, separate databases and distinct releases |
| 2C · Local safety | `packages/product-local/{client_linux.go,types.go}`, simulator transaction | Both products rejected a lock activated after preflight. Readiness, release version and product-specific healthy code were checked |
| 2D · Backup | `packages/local-policy/policy.go`, `tests/fixtures/{site.go,deployment.go}` | Signed MC-14, enrolled backup signer, expected dataset set, site, operation, backup-manifest digest, encryption, integrity and freshness |
| 2E · Reconciliation | `services/site-agent/{agent.go,deployment.go}`, `services/local-helper/{helper.go,deployment.go}`, `tests/sprint2/{central.go,workflow_linux_test.go}` | Original committed-disconnect witness passed. Two jobs produced two logical central receipts from three submission attempts |
| 2F · Failure handling | `tests/sprint2/failures_linux_test.go`, helper result and product fence | Before-effect rejection, after-effect failure, post-check failure and unknown outcome produced distinct signed results |
| 2G · Crash consistency | `tests/sprint2/crash_linux_test.go` with subprocess fixture in `workflow_linux_test.go` | CR-01–CR-10, including independent product death, expiry and repeated interrupted recovery |
| 2H · Boundary hardening | `tests/sprint2/{central_test.go,conflicts_linux_test.go,framing_linux_test.go}` and the admission suites | First-receipt binding, immutable results, unresolved holds, concurrency, replay and peer/framing negatives |
| 2I · Maintainability | Sprint 2 Go files, typed helper statuses, separate reservation and reconciliation paths | Standard Go formatting, Windows and Linux-target vet, explicit durable-boundary comments, bounded wire operations |

## Execution and authority

1. The loopback central simulator verifies the site-scoped MC-03, checks the enrolled installation and agent, and stores the signed bundle, artifact digest and expected execution identity. An exact resubmission returns the original queue entry.
2. The agent verifies MC-03, checks its clock and sequence, independently validates MC-05, MC-09 including site consent, and required MC-14. It queries both real LOCAL-01 endpoints. One SQLite transaction stores its consumed grants, pending bundle and expected execution ID before a helper request can be delivered.
3. The helper authenticates the peer and verifies the signed request against its own trust configuration. Its own SQLite transaction reserves the job, idempotency key, nonce, sequence, execution ID and grants. A lookup authenticates the scoped command and matches the entire previously reserved bundle. Creating an effect requires current deploy authority. Retrying a committed observation requires only authenticated reconciliation.
4. The simulated product accepts mutation only from the helper-capability client on the pinned socket. In one immediate transaction it checks current activity, readiness and the earliest applicable expiry, then writes either the effect and release revision or a durable no-effect fence. A duplicate finds that decision. The two transactions in the agent and helper each legitimately consume one grant.
5. The helper records its terminal result after observing the product and checking the expected release. The agent validates the result, signs MC-04 and commits it locally before delivery. Central verifies the signature, scope, expected execution, observed artifact and terminal state/stage/code. MC-04 carries no installation or agent fields. The simulator binds those through the signed dispatch target and its enrolled receipt signer. An acknowledgement marks the local outbox row delivered. Restart sends unacknowledged receipts before fetching new work.

| Product | Lock | Healthy code | Verified release transition | Required synthetic dataset |
|---|---|---|---|---|
| `zyposono-sim` | `ACTIVE_ACQUISITION` | `ACQUISITION_READY` | `sono-test-1` → `sono-test-2` | `APP_CONFIG` |
| `mock-secondary` | `BATCH_IN_PROGRESS` | `WORKER_READY` | `worker-test-4` → `worker-test-5` | `MANAGEMENT_AGENT` |

The test roles run as Linux UID/GID 65534. The agent opens a read-only product client and the helper receives the separate mutation capability. Tests rejected direct agent apply and fence calls, a wrong capability, and mismatched kernel credentials. Independently installed service accounts, privileged execution and resistance to compromise of the shared OS account remain outside this evidence.

## Crash-consistency matrix

Each row starts with its own databases. Columns A, H, P and C describe committed agent, helper, product and central state at interruption. “Queued” means central has the command and zero terminal receipts. Effects are shown at interruption → after reconciliation. Grants show agent/helper at interruption → final. In every row the final central store has exactly one logical terminal receipt **for the original job**. For CR-04 the preserved test also processes the second product.

**E** is the original execution ID `3f23f691-9aff-5c1b-83a2-64a2ce959097` for job `7da98fc5-a9c0-53c1-9958-d288d62b4f49`, product `zyposono-sim` and the digest of its signed manifest. Each new crash test checks the bundle binding in the agent and helper, the product identity in its own database, and the signed central receipt. In CR-01 the agent already knows E when the helper has no reservation. “Matrix” below means the named subtest of `TestCrashMatrix`.

| Case and barrier | A | H | P | C | ID | Effects | Grants A/H | Recovery and executed witness |
|---|---|---|---|---|---|---|---|---|
| CR-01 · before first helper lookup | Pending | Empty | Initial | Queued | E | 0 → 1 | 1/0 → 1/1 | New reservation, original job succeeds. Matrix/CR01 |
| CR-02 · before product lookup | Pending | Reserved | Initial | Queued | E | 0 → 1 | 1/1 → 1/1 | Resume original reservation. Matrix/CR02 |
| CR-03 · product apply returned to helper | Pending | Reserved | Applied | Queued | E | 1 → 1 | 1/1 → 1/1 | Recover from product ledger. Matrix/CR03 |
| CR-04 · real relay withholds committed helper response | Pending | Terminal | Applied | Queued | E | 1 → 1 | 1/1 → 1/1 | Kill agent and helper, recover original result. TestCommittedDisconnect |
| CR-05 · central pauses before accepting receipt | Signed, unacknowledged | Terminal | Applied | Queued | E | 1 → 1 | 1/1 → 1/1 | Deliver original outbox bytes. Matrix/CR05 |
| CR-06 · central accepts before ACK | Signed, unacknowledged | Terminal | Applied | Receipt | E | 1 → 1 | 1/1 → 1/1 | Redelivery acknowledged. Matrix/CR06 and TestCommittedDisconnect |
| CR-07 · product killed with management paused after apply | Pending | Reserved | Applied | Queued | E | 1 → 1 | 1/1 → 1/1 | Product reopened its DB, served the applied release, and accepted a synthetic local activity event while both management processes were down. Matrix/CR07 |
| CR-08 · kill after reservation, restart agent first | Pending | Reserved | Initial | Queued | E | 0 → 1 | 1/1 → 1/1 | First lookup released after helper rebind. Matrix/CR08. CR-02 exercises the reverse startup order |
| CR-09 · expiry after product commit | Pending | Reserved | Applied | Queued | E | 1 → 1 | 1/1 → 1/1 | Read original effect after job and grant expiry. Matrix/CR09-expired-committed |
| CR-09 · expiry before effect | Pending | Reserved | Initial | Queued | E | 0 → 0 | 1/1 → 1/1 | Durable fence and REJECTED receipt. Matrix/CR09-expired-reservation |
| CR-09 · stale backup or revoked site-consent key | Pending | Reserved | Initial | Queued | E | 0 → 0 | 1/1 → 1/1 | Separate runs fenced both original reservations. Matrix/CR09-stale-backup and Matrix/CR09-revoked-grant |
| CR-10 · two further disconnects while retrieving original result | Pending | Reserved, then terminal | Applied | Queued | E | 1 → 1 | 1/1 → 1/1 | Two recovering agents killed before accepting the real helper reply, third delivered it. Matrix/CR10 |

Barriers bracket real database commits and socket operations. The parent issues process kills and opens independent SQLite connections to inspect the records. Product lookup, delayed activation and conflicting execution ID probes run after restart. The matrix also checks that a different signed job cannot consume another grant while an original reservation is unresolved. [Crash witnesses](../../tests/sprint2/crash_linux_test.go) · [Original disconnect witness](../../tests/sprint2/workflow_linux_test.go)

## Admission, failure and recovery

- **54 admission negatives.** Each of 27 cases ran for both products, at the agent and independently at the helper. They cover tenancy, installation, purpose, revoked authority, artifact and product substitution, missing consent, wrong or extra datasets, wrong backup signer or manifest, stale/tampered evidence, encryption/integrity failure, unknown locks, offline/malformed LOCAL-01 and timeout. Each case checked zero grants consumed and zero product effects.
- **14 effect-boundary cases.** Each product saw a lock appear after preflight, job expiry, grant expiry and backup expiry at mutation, plus after-effect failure, post-effect unready and post-check unavailability. Rejections before activation produced zero effects. Post-effect failures preserved one. A subsequent signed deployment was held without consumption of its helper grant.
- **Concurrency and replay.** Twelve simultaneous execute requests returned one execution ID. Mixed lookup, execute and abort requests settled on the first durable decision. A changed bundle under the same job, reused idempotency key, nonce, sequence, or already consumed grant across products failed. The next independently authorised job remained admissible. A substituted execution ID from the other product produced zero agent receipts until the original helper result was retrieved.
- **Local transport.** Six malformed request cases and ten response/credential cases cover framing, size, truncation, duplicate keys, unexpected fields/status/results, cancellation or timeout, and wrong server UID/GID. The helper's kernel-peer suite separately checks allowed, wrong client UID and wrong client GID. Path tests preserve an occupied regular file and an active socket, reject a writable directory and refuse a symlink lock. Actual process restarts exercise stale owned sockets.
- **Central.** Correctly signed mismatched first receipts were rejected for scope, job, execution, signer, artifact, confidence and inconsistent terminal fields. Reopening the central database and redelivering preserved the original receipt. A pre-upgrade command with a random execution ID stays pending until its original trusted helper reservation is provisioned locally. Accepted historical receipts retain their original bytes.
- **Compensation.** ABSENT triggers fresh admission or refusal. PENDING and RETRY preserve the pending bundle. An unauthenticated or invalid request is denied. REFUSED is a durable pre-reservation refusal. DONE carries a validated result, which can record success, rejection, failure or an unknown verification outcome. A post-effect FAILED or UNKNOWN holds further deployments at the helper even after a terminal MC-04 was signed. The evidence and local application remain available for independently authorised investigation. No compensation or hold-release operation is authorised in this simulator.

## Defects corrected

| Baseline behavior | Correction and regression |
|---|---|
| First validly signed central receipt could supply an unrelated execution ID or inconsistent terminal state | Pin execution at dispatch and agent admission, pin enrolled target/signer, validate outcome tuple before first insertion. TestCentralReceiptBinding and TestSubstitutedHelperExecution |
| Completion of a failed or unknown post-check released the helper admission gate | Read the durable terminal state when reserving another deployment. TestEffectBoundaries |
| Backup admission could accept extra dataset categories or another same-purpose site signer | Check expected categories, enrolled authority and stated age against signed timestamps. TestAdmissionFailures |
| The product could treat an execution ID in the other effect/fence table as unused for another job | Check job and execution across both tables in the same write transaction. Reject nil and noncanonical UUIDs. TestLocalAuthority and crash recovery probes |
| Helper recovery could return a saved result without observing its trusted-clock floor | Persist/check the floor on authenticated recovery and compare the saved result to the original reservation. Existing clock regression and Linux reconciliation suites ran |
| Unexpected helper status or contradictory result could reach the agent, and socket replacement relied on directory ownership alone | Validate the reply shape, existing socket owner and lock file ownership/type. TestInvalidHelperReplies and TestSocketPathSafety |

## Verification commands and counts

Counts below use Go's named pass events, including parent tests and subtests. Platform builds are counted separately. Packages without test files are excluded. The Linux skip is `TestServiceProcess` during standalone discovery. The crash suite explicitly invokes that entry point in agent, helper and product children.


| Environment and command | Passed | Failed | Skipped |
|---|---:|---:|---:|
| Windows · `go test -json -count=1 ./...` | 58 | 0 | 0 |
| Windows · `go test -race -json -count=1 ./...` | 58 | 0 | 0 |
| WSL2 Linux amd64 · four ordinary test binaries | 179 | 0 | 1 |
| WSL2 Linux amd64 · four race-instrumented binaries | 179 | 0 | 1 |
| Windows · `pnpm.cmd test` | 31 | 0 | 0 |
| Windows · `pnpm.cmd test:interop` | 2 | 0 | 0 |
| Windows · PostgreSQL 16.13 integration groups | 5 | 0 | 0 |
| Windows · `pnpm.cmd lint`, `pnpm.cmd typecheck` | 2 commands | 0 | 0 |
| Windows and Linux target · `go vet ./...` | 2 commands | 0 | 0 |
| Scoped `gofmt -l`, scratch binding generation and `git diff --check` | Clean formatting, 2/2 bindings match, clean diff | 0 | 0 |

| Go package | Windows ordinary / race | Linux ordinary / race |
|---|---:|---:|
| `tests/cryptography` | 29 / 29 | 29 / 29 |
| `services/local-helper` | 15 / 15 | 19 / 19 |
| `services/site-agent` | 13 / 13 | 13 / 13 |
| `tests/sprint2` | 1 / 1 | 118 / 118, one standalone role skip per run |

The runner used Go 1.27.4, Node 24.14.0, pnpm 10.32.1 and Prisma 6.19.3. Command details follow. Toolchain and compiler caches live under ignored `.work/`. The Windows builds used the workspace Go toolchain. Windows race used the workspace LLVM MinGW `clang` and `CGO_ENABLED=1`.

```powershell
$env:GOCACHE="$pwd/.work/gocache"
$env:GOMODCACHE="$pwd/.work/gomod"
$env:Path="$pwd/.work/go/bin;"+$env:Path
go vet ./...
go test -json -count=1 ./... | Out-File -Encoding utf8 .work/s2-final/windows.jsonl
$env:Path="$pwd/.work/llvm-mingw-20261117-ucrt-x86_64/bin;"+$env:Path
$env:CGO_ENABLED='1'
$env:CC='clang'
go test -race -json -count=1 ./... | Out-File -Encoding utf8 .work/s2-final/windows-race.jsonl
pnpm.cmd lint
pnpm.cmd typecheck
pnpm.cmd test
pnpm.cmd test:interop
$env:PGBIN='C:\Program Files\PostgreSQL\16\bin'
node infrastructure/local-development/tenancy.mjs
```

For Linux, Go compiled the current tree on Windows. All four binaries then executed inside WSL2 `docker-desktop` as `nobody`, UID/GID 65534. The race builds used Zig 0.14.1 and a workspace-local glibc loader. The following PowerShell commands show the build flags and runtime command for each package.

```powershell
$env:GOOS='linux'
$env:GOARCH='amd64'
$env:CGO_ENABLED='0'
go vet ./...
go test -c -o .work/s2-final/linux/ ./...

$env:CGO_ENABLED='1'
$env:CC="$pwd/.work/zigtool/ziglang/zig.exe cc -target x86_64-linux-gnu.2.36"
$env:ZIG_GLOBAL_CACHE_DIR="$pwd/.work/zig-global"
$env:ZIG_LOCAL_CACHE_DIR="$pwd/.work/zig-local"
go test -race -c -ldflags '-linkmode=external -extldflags=-Wl,--dynamic-linker,/mnt/host/d/ZypoOps-One/.work/linux-glibc/ld-linux-x86-64.so.2,-rpath,/mnt/host/d/ZypoOps-One/.work/linux-glibc' -o .work/s2-final/linux-race/ ./...

wsl.exe -d docker-desktop -u nobody --exec id
foreach ($variant in 'linux','linux-race') {
  foreach ($suite in 'cryptography','local-helper','site-agent','sprint2') {
    wsl.exe -d docker-desktop -u nobody --exec "/mnt/host/d/ZypoOps-One/.work/s2-final/$variant/$suite.test" '-test.v' '-test.count=1'
  }
}
```

The final two product assertions also submit the original execution ID with a different job to both apply and fence and check both table counts. The affected package was rebuilt in each mode and `-test.run=^TestCrashMatrix$` was executed under the same WSL account. These targeted results are separate from the suite totals.

**Contract and source-pack checks.** The generator read copies of the catalog and schemas in `.work/s2-final/drift` via `node D:\ZypoOps-One\scripts\generate-types.mjs`. Both generated files matched the tracked copies after CRLF normalization. `git diff --exit-code -- packages/management-contracts` returned zero. `gofmt -l` covered the 20 Sprint 2 Go paths from the baseline plus the three new Linux tests.

| Additional regression command | Observed pass |
|---|---|
| `python .work/pack/tests/validate_pack.py` | 15/15 schema fixtures, 9/9 command shapes, synthetic CSR/HMAC and negative structural checks |
| `python .work/pack/tests/local_api_contract_tests.py` | 5/5 LOCAL-01 shapes and forbidden-field rejection |
| `python .work/pack/tests/sqlite_agent_tests.py` | One harness covering SQLite DDL, replay, key rotation, receipt/progress and integrity |
| `node .work/pack/tests/verify_vectors.mjs` | 15/15 signed fixtures, two signed negatives and parser checks |
| `go run .work/pack/tests/verify_vectors.go .work/pack` | 15/15 historical ASCII fixtures, two signed negatives and duplicate-key rejection |

Those five harness commands exited zero. Python was the bundled Codex runtime with `PYTHONPATH=.work/pythonlibs`. The cross-runtime suite separately exercised TS → Go → TS for ES256 and Ed25519. The Sprint 1 Go regression included the 100-iteration SQLite crash loop, transaction-boundary child kills, boot-ID change, trusted-clock rollback/recovery, key rotation, multi-grant atomicity and receipt replay.

**PostgreSQL concurrency.** The isolated cluster applied `001_o0` and `002_runtime`. A second Prisma deployment found zero pending migrations. A Nest application submitted 120 concurrent Prisma transactions split across two tenant/site identities through `connection_limit=4`. Each transaction inserted one health row and saw only its expected site. The independent administrative count was 120. An unscoped runtime read returned zero sites. Two competing approval-consumption attempts produced one winner. Runtime and migration identities were separate, and the cluster shut down after the five test groups passed. [Executable concurrency assertions](../../tests/tenancy/isolation.test.ts) · [Cluster harness](../../infrastructure/local-development/tenancy.mjs)

## Handoff and governance

The exercised paths have no known failing test. A deployment held after FAILED or UNKNOWN requires separate recovery authority. A pre-upgrade random-ID reservation requires trusted historical evidence for first central acceptance. LOCAL-01 service-account separation, production trust/custody, a privileged installation, real OCI effects, physical power-loss behavior and clinical acceptance remain outside the synthetic scope. Hosted CI for the local work requires a separately authorised commit and push.

OD01–OD08 remain proposed and unsigned. Human decisions still cover clinical independence, protected agent identity, PKI/enrolment, signing and site consent, operator IAM/tenancy, product lifecycle and joint local-interface approval, privacy/retention, and offline continuity. Q01–Q07 and independent O0 findings retain their existing owners. The independent O0 production NO_GO remains in force.

**Next engineering step:** Submit the Sprint 2 working diff, checkpoint and this evidence to independent Sprint 2 review. Sprint 3 can be considered after that review accepts the evidence and resolves its findings. Production trust integration and clinical release retain their separate approval gates.

Branch `main` remains at the resume commit. The working change set comprises 20 modified Sprint 2 Go files, three new Linux test files, and the checkpoint and implementation report. No commit, push or deployment occurred. [Resume checkpoint](O1-Sprint-2-Checkpoint.md)


## Final verdict

```text
O1_SPRINT_2_STATUS: COMPLETE_TEST_ONLY
RESUME_BASELINE_COMMIT: 135fb6619bc95401263c0b184456cea4615f03de
EXISTING_IMPLEMENTATION_PRESERVED: YES
AGENT_HELPER_SOCKET_INTEGRATION: PASS_LINUX
TWO_PRODUCT_SIMULATION: PASS
LOCAL_01_MAINTENANCE_LOCKS: PASS_BOTH_PRODUCTS
MC14_BACKUP_EVIDENCE: PASS_BOTH_PRODUCTS
POST_COMMIT_DISCONNECT_RECOVERY: PASS
PRODUCT_RESTART_RECOVERY: PASS
CROSS_LEDGER_CRASH_CONSISTENCY: PASS_CR01_TO_CR10
EXPIRED_AUTHORISATION_RECONCILIATION: PASS
DUPLICATE_EXECUTION_PREVENTION: PASS
APPROVAL_CONSUMPTION_INTEGRITY: PASS_BOTH_LEDGERS
UNRESOLVED_DEPLOYMENT_PROTECTION: PASS_SITE_HOLD
DURABLE_RECEIPT_RECONCILIATION: PASS
CENTRAL_RECEIPT_IDENTITY_BINDING: PASS_TEST_ENROLMENT
SECURITY_NEGATIVE_TESTS: PASS_TESTED_CASES
REGRESSION_TESTS: PASS_LOCAL
GO_RACE_TESTS: PASS_WINDOWS_AND_LINUX
POSTGRESQL_TESTS: PASS_5_OF_5
LINUX_RUNTIME_TESTS: PASS_UNPRIVILEGED_WSL2
HOSTED_CI_BASELINE: PASS_RUN_45427502336
HOSTED_CI_CURRENT_CHANGES: NOT_RUN_UNCOMMITTED
OD01_TO_OD08_APPROVED: 0_OF_8
PRODUCTION_TRUST_INTEGRATION_ALLOWED: NO
O1_RUNTIME_VERIFIED: NO_FORMAL_GATE_PASS
CLINICAL_PRODUCTION_READY: NO_GO
READY_FOR_INDEPENDENT_SPRINT_2_AUDIT: YES
READY_FOR_O1_SPRINT_3: NO_PENDING_INDEPENDENT_REVIEW
```

## Dated status correction · 25 November 2026

The previously local Sprint 2 candidate was subsequently committed as `8f9d88be2013c9b014202fc2435ec1368d83575c`. Hosted run 45594545137 passed on that SHA. The earlier entries above record the state when the original regression ran. The independent audit returned GO_WITH_CONDITIONS_TEST_ONLY. Sprint 3 awaits independent remediation acceptance. [Hosted run](https://github.com/zyposofttech/ZypoOps-One/actions/runs/45594545137)

AUD-S2-P3-01 document erratum. The Operations Platform Master BRD v1.2 carries stale v1.1 footer text. Use the title, revision history and recorded source checksum to identify the revision. The original PDFs and their checksums are preserved.
