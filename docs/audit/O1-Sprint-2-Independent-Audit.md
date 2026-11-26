# ZypoOps One — Independent O1 Sprint 1 & Sprint 2 Security, Architecture and Runtime Audit

**Audit Date:** 25 November 2026  
**Audited Repository:** `https://github.com/zyposofttech/ZypoOps-One` (branch `main`, source archive snapshot)  
**Audited Commit SHA:** `8f9d88be2013c9b014202fc2435ec1368d83575c` (`O1-Sprint 2 Completed`)  
**Previous Sprint 2 Baseline:** `135fb6619bc95401263c0b184456cea4615f03de`  
**Audit Role:** Independent Principal Software Architect, Distributed Systems Reliability Engineer, Go Security Engineer, PostgreSQL Security Specialist, Application Security Auditor & Clinical Infrastructure Safety Reviewer  
**Audit Mode:** Read-only independent audit (no tracked application source, contracts, specifications, or prior implementation reports modified; no Sprint 3 work performed)

---

## 1. Executive Verdict

- **Independent O1 Sprint 2 Engineering Verdict (Test-Only Scope):** **`GO_WITH_CONDITIONS_TEST_ONLY`**
- **O0 Production Gate:** **`NO_GO`** (0/8 human approvals on `OD01–OD08`; `Q01–Q07` open; joint `LOCAL-01` sign-off open)
- **Clinical Production Readiness:** **`NO_GO`**
- **Ready for O1 Sprint 3 (Test-Only):** **`CONDITIONAL`** (strictly gated on closing 2 blocking P1 conditions below)

### Summary of Independent Findings

| Severity | Count | Finding IDs | Sprint 3 Blocking? | Production Blocking? |
|---|---:|---|---|---|
| **P0 (Critical)** | 0 | None | No | No |
| **P1 (High)** | 2 | `AUD-S2-P1-01`, `AUD-S2-P1-02` | **Yes (2 conditions)** | **Yes** |
| **P2 (Medium)** | 3 | `AUD-S2-P2-01`, `AUD-S2-P2-02`, `AUD-S2-P2-03` | No | Yes |
| **P3 (Low)** | 3 | `AUD-S2-P3-01`, `AUD-S2-P3-02`, `AUD-S2-P3-03` | No | No |
| **GOV (Governance / Production Gate)** | 6 | `AUD-S2-GOV-01` through `AUD-S2-GOV-06` | No (test-only) | **Yes (Hard Stop)** |

The integrated O1 Sprint 1 and Sprint 2 codebase at `8f9d88be2013c9b014202fc2435ec1368d83575c` demonstrates strong architectural discipline, strict product neutrality, sound cryptographic domain separation, well-constructed PostgreSQL 16 RLS policies, bilateral Linux `SO_PEERCRED` verification, and an atomic product-side final safety transaction. **No P0 security bypass, cross-tenant leak, duplicate committed effect, or clinical-lock bypass was found in source inspection.**

However, unconditional `GO_TEST_ONLY` cannot be issued for two concrete, independently verified reasons:
1. **`AUD-S2-P1-01` (Code Defect — CR-09 Key-Revocation Reconciliation Wedge):** Both `Agent.Handle()` (`services/site-agent/agent.go:159`) and `Helper.dispatch()` (`services/local-helper/deployment.go:46`) unconditionally verify `MC-03` against the live trust store *before* checking for an already-admitted `pending_deployment` / `deployment_execution` row. If the `ops_job` key (`mock-job`) is revoked after CR-02 (helper reservation) or CR-03 (product effect committed), `Agent.Run()` wedges on every restart with `UNTRUSTED_KEY` and `Helper.dispatch()` returns `DENIED`, preventing both fencing of unapplied reservations and reconciliation/receipt signing of already-committed effects. `TestCrashMatrix/CR09-revoked-grant` only revoked `mock-local` (`site_authoriser`), leaving `ops_job` revocation uncovered.
2. **`AUD-S2-P1-02` (Independent Runtime Execution Gap in Audit Sandbox):** The audit environment received a GitHub source snapshot archive (no `.git/` metadata, no `node_modules/`), Linux 6.12.105+, Node v18.20.4, and Python 3, without a Go 1.27.4 compiler, PostgreSQL 16 binaries, or access to private GitHub Actions runs (`45427502336`, `45594545137`). Independent execution verified contract-binding drift (zero diff), all 15/15 golden + 2/2 negative + adversarial cross-site MC-09 cryptographic vectors in Node, and 6/6 Agent SQLite WAL/FK/uniqueness constraints in Python. Go tests (ordinary and `-race`), Linux `SO_PEERCRED`/CR-01..CR-10 process crash tests, TypeScript/interop tests, PostgreSQL 16 120-tx RLS tests, and hosted CI SHA attribution remain **`IMPLEMENTED_UNVERIFIED`** from this independent audit environment until executed in an equipped Linux + Go 1.27.4 + Node 24 + PG16 runner or verified against signed CI logs for `8f9d88be2013c9b014202fc2435ec1368d83575c`.

---

## 2. Exact Audited Commit & Immutable Baseline

| Baseline Attribute | Observed Independent Value | Evidence Classification |
|---|---|---|
| Repository | `https://github.com/zyposofttech/ZypoOps-One` | `VERIFIED_SOURCE` |
| Source Revision Under Audit | `8f9d88be2013c9b014202fc2435ec1368d83575c` (archive root `ZypoOps-One-8f9d88be2013c9b014202fc2435ec1368d83575c`) | `VERIFIED_SOURCE` |
| Git Branch / HEAD / Working Tree | Source snapshot ZIP provided without `.git/` directory; git history, staged/unstaged diff, and branch pointer could not be queried locally | `PARTIALLY_VERIFIED` |
| Tracked Specification Checksums (`docs/`) | 5/5 tracked primary specifications in `docs/` match `docs/implementation/source-sha256.txt` byte-for-byte; 3 revised reading copies in `docs/implementation/specifications/` differ as documented in `O1-Sprint-1.md` | `VERIFIED_RUNTIME` |
| Hosted CI Run `45427502336` (Initial Sprint 2) | Private repository URL inaccessible to audit web browser (`401/login`); SHA attribution (`135fb6619bc95401263c0b184456cea4615f03de`) unverified externally | `IMPLEMENTED_UNVERIFIED` |
| Hosted CI Run `45594545137` (Hardened Sprint 2) | Private repository URL inaccessible to audit web browser (`401/login`); SHA attribution (`8f9d88be2013c9b014202fc2435ec1368d83575c`) unverified externally | `IMPLEMENTED_UNVERIFIED` |
| Audit Sandbox Environment | Linux `6.12.105+` x86_64, Node `v18.20.4`, npm `9.2.0`, Python 3 (`sqlite3`), `git`; no Go toolchain, no PostgreSQL 16, no `pnpm`/`node_modules`, no outbound internet | `VERIFIED_RUNTIME` |

---

## 3. Audit Scope and Exclusions

**In Scope (Authorised Test-Only Engineering Verification):**
- Integrated O1 Sprint 1 (Waves 1A–1D) and O1 Sprint 2 (Waves 2A–2I) source code, schemas, migrations, generated bindings, fixtures, CI workflow (`.github/workflows/foundation.yml`), ADRs (`OD01–OD08`), and implementation reports.
- Architecture and product neutrality, contracts and cryptography, PostgreSQL/Prisma RLS design, Site Agent durability and replay, Local Helper trust boundary and Linux Unix sockets, `LOCAL-01` maintenance safety, `MC-14` backup evidence, CR-01..CR-10 crash consistency, concurrency/fencing, central dispatch/receipt reconciliation, unresolved deployment holds, privacy/governance, and test integrity.

**Explicit Exclusions (Separately Gated Production Requirements — Not Defects in Test-Only Scope):**
- Production OS service-account separation (separate systemd users/groups for Agent, Helper, and Clinical Application) and compromised-OS-account resistance.
- Hardware TPM 2.0 key custody, production CA/KMS (`OD03`/`Q01`), OIDC IdP / phishing-resistant MFA (`OD05`/`Q05`), TUF anti-rollback/freeze metadata (`OD04`/`Q04`), and production mTLS transport.
- Real OCI/Podman container execution, real database migration execution, physical power-loss durability testing, authorised hold-release/compensation workflows, and clinical commissioning.

---

## 4. Architecture Assessment (Domain A — Architecture & Product Neutrality)

**Score:** 9.0 / 10 | **Evidence:** `VERIFIED_SOURCE`

- **Central vs. Local Clinical Authority Separation:** Central Ops (`services/ops-api/app.ts`, `packages/database/`, `tests/sprint2/central.go`) contains zero patient-data tables, zero DICOM/SR fields, and zero clinical database credentials. Clinical/local safety authority stays exclusively with the simulated product (`products/simulator/application_linux.go`).
- **Outbound-Only Management Transport:** `Agent.Run()` (`services/site-agent/agent.go:296-342`) initiates all central communication outbound (`GET /next`, `POST /receipt`) with no inbound listener on the Agent.
- **Least-Privilege Local Boundaries:** Site Agent receives a read-only `productlocal.Client` (empty `WriteToken`), while only Local Helper holds the mutation capability (`WriteToken` checked via `subtle.ConstantTimeCompare` in `products/simulator/application_linux.go:114`). Neither Agent nor Helper invokes `os/exec`, shell commands, arbitrary filesystem mounts, or clinical database drivers (`grep` confirmed zero `os/exec` outside test harnesses).
- **Genuine Product Neutrality:** Both `zyposono-sim` (`products/zyposono-sim/profile_linux.go`) and `mock-secondary` (`products/mock-secondary/profile_linux.go`) run against the exact same compiled Agent (`services/site-agent/`) and Helper (`services/local-helper/`). Product-specific locks (`ACTIVE_ACQUISITION` vs. `BATCH_IN_PROGRESS`), healthy service codes (`ACQUISITION_READY` vs. `WORKER_READY`), release identifiers (`sono-test-1->2` vs. `worker-test-4->5`), and synthetic backup dataset categories (`APP_CONFIG` vs. `MANAGEMENT_AGENT`) are externalised in profile configuration and SQLite databases (`<product>.db`). Zero ZypoSono-specific logic exists in `packages/` or `services/`.
- **Product Substitution Prevention:** Product identity is cryptographically bound across `MC-03` (`params.product_key`), `MC-05` (`body.product_id`), `MC-09` (`body.resource_scope`), `TestArtifacts[digest]`, `BackupForJob[job_id].Product`, and the product simulator's own `application.product` check (`products/simulator/application_linux.go:49,131`).

---

## 5. Sprint 1 Regression Assessment

- **Wave 1A (Contracts & Cryptography):** `VERIFIED_RUNTIME` (contract drift, golden/negative vectors, cross-site MC-09 probe) / `IMPLEMENTED_UNVERIFIED` (Go/TS test runner & interop bridge execution). `scripts/generate-types.mjs` reproduced `packages/management-contracts/generated/types.ts` and `packages/management-contracts/types_generated.go` with **zero diff**.
- **Wave 1B (Database & RLS):** `VERIFIED_SOURCE` / `IMPLEMENTED_UNVERIFIED` (live PG16 execution unavailable in sandbox). Both `001_o0` and `002_runtime` migrations, `packages/database/tenant.ts`, `services/ops-api/app.ts`, `infrastructure/local-development/tenancy.mjs`, and `tests/tenancy/isolation.test.ts` remain intact with zero regressions.
- **Wave 1C (Site Agent Durability & Clock Recovery):** `VERIFIED_SOURCE` + `VERIFIED_RUNTIME` (SQLite DDL/PRAGMA/constraint probes) / `IMPLEMENTED_UNVERIFIED` (Go test execution). Watermark preservation across rejected jobs, boot-ID invalidation, loopback TLS pinned-certificate `RecoverClock()`, and replay protection remain intact.
- **Wave 1D (Local Helper Prototype):** `VERIFIED_SOURCE` / `IMPLEMENTED_UNVERIFIED` (Go Linux socket execution). Sprint 1 `apply()` path and `socket_linux_test.go` remain intact, though `AUD-S2-P2-03` notes that hardened `BackupForJob` checks only run when `LocalCheck != nil`.

---

## 6. Sprint 2 Waves 2A–2I Assessment

| Wave | Scope | Source Audit | Runtime Audit (This Sandbox) | Key Findings / Notes |
|---|---|---|---|---|
| **2A** | Agent–Helper Unix Socket & Bilateral `SO_PEERCRED` | Strong | `IMPLEMENTED_UNVERIFIED` | `AUD-S2-P2-01` (serial accept loop); strong `O_NOFOLLOW`/`Flock`/dir perm/owner checks |
| **2B** | Two-Product Neutrality (`zyposono-sim`, `mock-secondary`) | Strong | `IMPLEMENTED_UNVERIFIED` | Genuine product-neutral core; independent DBs, sockets, locks, healthy codes |
| **2C** | `LOCAL-01` Maintenance Locks & Readiness | Strong | `IMPLEMENTED_UNVERIFIED` | Atomic `BEGIN IMMEDIATE` product-side final lock/readiness/expiry check prevents TOCTOU |
| **2D** | `MC-14` Backup Evidence Validation | Strong (with P2 caveat) | `VERIFIED_RUNTIME` (schema/vector) / `IMPLEMENTED_UNVERIFIED` (Go) | `BackupForJob` independently configured on Agent & Helper; `AUD-S2-P2-03` (`LocalCheck != nil` gate) |
| **2E** | Disconnect & Receipt Reconciliation | Strong | `IMPLEMENTED_UNVERIFIED` | Durable pending table, deterministic execution ID, outbox redelivery before `Next()` |
| **2F** | Failure Handling & Unresolved Holds | Strong | `IMPLEMENTED_UNVERIFIED` | Distinct `BEFORE_EFFECT`/`AFTER_EFFECT`/`POST_VERIFY`/`UNKNOWN`; durable site-wide hold on helper |
| **2G** | Distributed Crash Consistency (CR-01..CR-10) | **P1 Gap on CR-09** | `IMPLEMENTED_UNVERIFIED` | `AUD-S2-P1-01`: revoking `ops_job` key wedges pending reconciliation/fencing |
| **2H** | Security & Boundary Hardening | Strong | `VERIFIED_RUNTIME` (crypto/SQLite) / `IMPLEMENTED_UNVERIFIED` (Go) | 54 admission negatives, 14 effect-boundary cases, 16 framing/reply negatives, central binding |
| **2I** | Maintainability & Documentation | Minor gaps | `VERIFIED_SOURCE` | `AUD-S2-P3-01` (stale README & Sprint 2 uncommitted claims), `AUD-S2-P3-02` (`job_stage` UNKNOWN) |

---

## 7. Crash-Consistency Verification (Domain H — Highest Priority)

**Score:** 7.5 / 10 (capped by `AUD-S2-P1-01` and unexecuted Linux runtime in this sandbox)

### Boundary-by-Boundary Source Audit of CR-01 through CR-10

| Boundary | Barrier Location & Mechanism | Pre-Kill State Verified in Test | Post-Recovery Invariant Verified in Test | Independent Assessment |
|---|---|---|---|---|
| **CR-01** | `witnessSocket.Exchange` (`pending`) before first Helper `lookup` | `agent.db`: pending=1, grants=1, receipt=0; `helper.db`: grants=0; `product.db`: effects=0 | Re-execs helper, 1 effect, 1/1 grants, 1 central receipt with pinned `E` | `IMPLEMENTED_UNVERIFIED` — barrier genuinely brackets Agent commit before Helper IPC |
| **CR-02** | `witnessProduct.Lookup` (`reserved`) after Helper reservation commit | `agent.db`: pending=1, grants=1; `helper.db`: grants=1, exec=`E`, result=NULL; `product.db`: effects=0 | Resumes original reservation, 1 effect, 1/1 grants, 1 receipt | `IMPLEMENTED_UNVERIFIED` — barrier brackets Helper reservation commit before product IPC |
| **CR-03** | `witnessProduct.Apply` (`applied`) after product `APPLIED` commit | `helper.db`: result=NULL; `product.db`: effects=1 (`Release`) | Helper recovers committed effect from product ledger without re-applying; 1 effect, 1 receipt | `IMPLEMENTED_UNVERIFIED` — genuine cross-ledger recovery |
| **CR-04** | `TestCommittedDisconnect` real Unix relay withholds Helper `DONE` reply | `helper.db`: result committed; `agent.db`: receipt=0; `product.db`: effects=1 | Both Agent & Helper killed and restarted; Agent recovers original `Execution` and signs MC-04 | `IMPLEMENTED_UNVERIFIED` — real socket relay, zero DB recreation |
| **CR-05** | Central HTTP handler pauses (`presend`) before `central.ServeHTTP` | `agent.db`: receipt=1 (unacked); `central.db`: terminal=0 | Restarted Agent delivers exact original outbox bytes; `central.db` terminal=1 | `IMPLEMENTED_UNVERIFIED` — verifies outbox immutability |
| **CR-06** | Central HTTP handler runs `central.ServeHTTP` then pauses (`preack`) / hijacks conn | `agent.db`: receipt=1 (unacked); `central.db`: terminal=1 | Redelivery accepted idempotently; Agent marks `acknowledged_at`; still 1 logical receipt | `IMPLEMENTED_UNVERIFIED` — verified in both `TestCrashMatrix/CR06` and `TestCommittedDisconnect` |
| **CR-07** | Product killed after `applied`, restarted with `SIGUSR1` local activity while Agent/Helper dead | Product serves applied release and writes `active=1, ready=1` during management outage | Local clinical continuity proven; management later reconciles original effect | `IMPLEMENTED_UNVERIFIED` — independent product process restart verified in source |
| **CR-08** | Agent restarted before Helper (`agentFirst=true`) after `reserved`; reverse order in CR-02 | Agent pauses at outbound call until Helper rebinds socket | Clean reconciliation in both startup orders | `IMPLEMENTED_UNVERIFIED` — covers both restart orders |
| **CR-09** | 4 subcases: `expired-committed`, `expired-reservation`, `stale-backup`, `revoked-grant` (`mock-local`) | Expired committed reconciles (1 effect); expired/stale/revoked reservation fences (0 effects, `REJECTED`) | **GAP (`AUD-S2-P1-01`):** Revoking `ops_job` (`mock-job`) after CR-02/CR-03 permanently wedges `Agent.Run()` and returns `DENIED` at Helper | **`FAILED` (partial — `ops_job` revocation gap)** |
| **CR-10** | 2 further kills at `reply` barrier during recovery before 3rd Agent delivers | Original reservation/effect preserved across 3 recovery attempts | 1 effect, 1/1 grants, 1 receipt with original `E` | `IMPLEMENTED_UNVERIFIED` — genuine repeated disconnect coverage |

**Database Isolation Check:** `TestServiceProcess` (`tests/sprint2/workflow_linux_test.go:176-258`) reopens existing `agent.db`, `helper.db`, and `<product>.db` from `ZOP2_DIR` without recreating databases or regenerating execution IDs. Post-recovery probes attempt delayed `Apply`, cross-job `Fence`/`Apply` with original `execution`, and altered `execution` on original `job`, asserting `effect` + `no_effect` counts remain invariant.

---

## 8. Cryptographic and Authorisation Assessment (Domain B & G)

**Score:** 9.0 / 10 (Crypto) | 8.5 / 10 (Backup Evidence) | **Evidence:** `VERIFIED_RUNTIME` (Node crypto + drift) & `VERIFIED_SOURCE`

- **RFC 8785 Canonicalisation & Strict Decoding:** Go (`packages/management-contracts/contract.go:39-49`) uses `go-json-experiment/json/jsontext` (`Value.Canonicalize()`) with depth limit 64, 1 MiB limit, and `encoding/json` numeric overflow rejection. TypeScript (`packages/management-contracts/json.ts:12-76`) implements a custom wire parser rejecting duplicate keys, lone surrogates (`!isWellFormed()`), non-finite numbers, nesting > 64, trailing tokens, and payloads > 1 MiB before JCS serialization.
- **Independent Verification Results:** In this audit, an independent Node `node:crypto` verifier confirmed **15/15 golden signed vectors** (exact `input_b64` match, Ed25519 + ES256/P-256 IEEE-P1363, nested `MC-09` `ZOP-SITE-CONSENT-V1` and `MC-12` `ZOP-CLINICAL-PACKAGE-V1`), **2/2 negative vectors rejected**, and **adversarial cross-site MC-09 re-signing rejected** (`BAD_SIGNATURE` even when attacker re-signs outer `MC-09` with a valid `ops_approval` key and clones the `site_authoriser` key to the target site).
- **Key-Purpose & Scope Enforcement:** Both `signing-go` and `signing-ts` enforce exact `kid`, `!revoked`, `alg`, `purpose`, and mandatory `(tenant_id, site_id)` scoping on `agent_*` and `site_authoriser` keys. An independently valid signature with the wrong key purpose cannot bypass any verification boundary.
- **Backup Evidence (`MC-14`):** Independently verified by both Agent and Helper (`packages/local-policy/policy.go:146-168`). Expected product, artifact digest, backup manifest digest, enrolled backup signer `kid`, and exact dataset categories originate from independently provisioned `Config.BackupForJob[job_id]`, not caller-controlled input. `MC-14.schema.json` enforces `uniqueItems: true` on `dataset_categories`. **Caveat (`AUD-S2-P2-03`):** `BackupForJob` checks are skipped when `c.LocalCheck == nil` (Sprint 1 legacy `Preflight` path).

---

## 9. Tenant-Isolation Assessment (Domain C)

**Score:** 9.0 / 10 (Source Design) | **Evidence:** `VERIFIED_SOURCE` / `IMPLEMENTED_UNVERIFIED` (no PG16 binary in sandbox)

- **Migrations (`001_o0` + `002_runtime`):** 31 tenant/site-scoped tables have `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY` with policies checking `app.tenant_id` and `app.site_id` (`operator_role` allows `scope_site_id IS NULL OR scope_site_id = app.site_id`). Global tables (`deployment_artifact`, `trusted_signing_key`) are explicitly separated from site-scoped `site_signing_key`.
- **Composite Foreign Keys & Triggers:** Composite `(tenant_id, site_id, id)` FKs prevent cross-tenant/sibling-site references. `zop.guard_job_transition()` enforces state machine transitions (`RECEIVED -> VALIDATED -> ACCEPTED -> RUNNING -> terminal`); `zop.consume_grant()` (`SECURITY DEFINER SET search_path=pg_catalog`, `REVOKE ALL FROM PUBLIC`) atomically enforces single-use grant consumption matching tenant, site, operation, `target_sha256`, validity window, and job expiry.
- **Role Separation & Prisma Pooling:** `infrastructure/local-development/tenancy.mjs` creates separate `zop_migrate` (table owner) and `zop_runtime` (`NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS`) roles with minimal DML grants (no `UPDATE`/`DELETE` on `operational_audit_event` or `job_receipt`). `packages/database/tenant.ts` validates UUID format and calls parameterized `set_config('app.tenant_id', ..., true)` and `set_config('app.site_id', ..., true)` inside `this.#client.$transaction(...)` (transaction-local, safe across pooled connections).
- **Test Design Review (`tests/tenancy/isolation.test.ts`):** Covers both read and write isolation (unscoped reads return `[]`, unscoped writes reject, sibling-site and cross-tenant writes reject, 120 concurrent Nest/Prisma transactions over `connection_limit=4` insert 120 health rows and read only their own site, concurrent grant consumption yields 1 winner). Production IAM/ingress remains gated (`AUD-S2-GOV-05`).

---

## 10. Local Transport, Maintenance Safety, Concurrency & Central Reconciliation (Domains D, E, F, I, J, K)

- **Agent Durability (Domain D — 8.5/10):** `agent_sqlite_v1_2.sql` enforces `WAL`, `synchronous=FULL`, `foreign_keys=ON`, and unique constraints on `(installation_id, command_type, idempotency_key)`, `(installation_id, sequence_epoch, sequence)`, `(installation_id, nonce)`, and `approval_consumption(grant_id)` — **independently verified in Python `sqlite3` (6/6 constraint rejections)**. Sequence floor survives key rotation; clock watermark survives rejected jobs via `SAVEPOINT`. **Note (`AUD-S2-P2-02`):** `admit()` holds a SQLite `BEGIN IMMEDIATE` transaction across Unix-socket `LOCAL-01` HTTP calls.
- **Helper & Socket Boundary (Domain E — 8.0/10):** Bilateral `SO_PEERCRED` (`packages/unixpeer/peer_linux.go:19-48`), directory ownership & `0022` write-bit rejection, `O_NOFOLLOW` + `Flock` lock file, stale socket ownership/active-dial verification, 1 MiB request cap, 4 KiB reply cap, duplicate/unknown JSON field rejection, and 4s timeouts. **Shared UID/GID Limitation:** All simulation roles run as UID/GID `65534`; this proves socket protocol and peer-check plumbing, **not** OS-level service-account isolation or compromised-agent containment (`AUD-S2-GOV-03`).
- **`LOCAL-01` Maintenance Safety (Domain F — 9.0/10):** Both `GET /local/v1/locks` and `GET /local/v1/readiness` are validated against `LOCAL-01` JSON Schema (`packages/product-local/client_linux.go:59-95`). Unknown locks, `lease_ttl_s <= 0`, contradictory `locked=false` with non-empty `reasons`, malformed JSON, timeouts, and unavailable sockets all fail closed. **TOCTOU Analysis:** `products/simulator/application_linux.go:156-205` performs the final lock (`active`), readiness (`ready`), and earliest-expiry (`effectDeadline`) check inside the same `BEGIN IMMEDIATE` transaction that writes `effect` and updates `application.version`, eliminating the TOCTOU window between preflight and simulated mutation.
- **Concurrency & Fencing (Domain I — 7.5/10):** Product-side cross-table `effect UNION ALL no_effect` check prevents execution-ID reuse across tables and jobs. **Limitation (`AUD-S2-P2-01`):** `Helper.Serve()` (`services/local-helper/socket_linux.go:33`) calls `handleConn()` synchronously, serializing all 12 concurrent requests in `TestConcurrentDuplicates` and all 5 requests in `TestCompetingOperations` at the kernel socket backlog rather than exercising concurrent handler goroutines.
- **Central Dispatch & Receipt Reconciliation (Domain J — 8.0/10):** `tests/sprint2/central.go` pins deterministic `ExecutionID(binding)` at dispatch, validates `MC-04` signature/scope/artifact/signer/outcome tuple on first receipt, preserves immutable receipt bytes across restarts, and supports explicit local `restoreLegacyReservation()` for pre-upgrade random IDs. **Contract Limitation:** Because `MC-04` omits `installation_id`, `agent_id`, and helper attestation, central trusts the enrolled `agent_receipt` key for outcome truthfulness; `central.go` also hardcodes single-site test constants (`AUD-S2-P3-03`).
- **Failure, Recovery & Unresolved Holds (Domain K — 8.5/10):** Clean semantic distinction across `ABSENT`, `PENDING`, `RETRY`, `REFUSED`, `DENIED`, `DONE`, `REJECTED` (`BEFORE_EFFECT`), `FAILED` (`AFTER_EFFECT` / `POST_VERIFY`), `UNKNOWN` (`POST_VERIFY`), and `SUCCEEDED` (`VERIFIED`). Helper holds the site durably across restarts whenever any `deployment_execution` row has `result IS NULL` or `state NOT IN ('SUCCEEDED','REJECTED')` (`services/local-helper/deployment.go:128-130`), blocking other products and jobs without consuming new grants. Minor stage-label defect in `Agent.dispatch()` (`AUD-S2-P3-02`).

---

## 11. Privacy & Clinical-Independence Assessment (Domain L)

**Score:** 9.5 / 10 | **Evidence:** `VERIFIED_SOURCE`

- Zero real patient records, zero DICOM/SR payloads, zero clinical database drivers/credentials, zero remote shell/arbitrary command execution, and zero production signing-key material anywhere in the repository.
- Deterministic test keys in `tests/fixtures/site.go:33` are explicitly labelled `"PUBLIC deterministic TEST fixture only/"` and isolated under `tests/fixtures/`.
- All `OD01–OD08` ADRs explicitly remain unsigned proposals (`0/8` approved) with no auto-approval or unsupported clinical readiness claims.

---

## 12. Test Integrity & Reconciliation Assessment (Domain M)

**Score:** 8.0 / 10 | **Evidence:** `VERIFIED_SOURCE` & `VERIFIED_RUNTIME` (static inventory)

Static inspection of every `*_test.go` and `*.test.ts` file independently **reconciled every single test count** reported in `docs/implementation/O1-Sprint-2.md` with **zero inflation or discrepancy**:

| Test Suite | Reported Count (`O1-Sprint-2.md`) | Independently Reconciled Static Structure | Sandbox Execution Status |
|---|---:|---|---|
| Windows Go pass events (`go test ./...`) | 58 (0 skip) | 29 crypto + 15 helper + 13 agent + 1 central = **58** | `IMPLEMENTED_UNVERIFIED` (no Go) |
| Linux Go pass events (4 packages) | 179 (1 skip) | 29 crypto + 19 helper + 13 agent + 118 sprint2 = **179** (+1 `TestServiceProcess` skip) | `IMPLEMENTED_UNVERIFIED` (no Go) |
| TypeScript contract tests | 31 | 1 + 15 + 2 + 1 + 1 + 1 + 1 + 9 = **31** | `IMPLEMENTED_UNVERIFIED` (no `node_modules`) |
| Cross-runtime crypto interop | 2 | ES256 + Ed25519 = **2** | `IMPLEMENTED_UNVERIFIED` (no Go/deps) |
| PostgreSQL 16 integration groups | 5 (120 pooled tx) | 5 `test(...)` blocks in `isolation.test.ts` (120 tx over `connection_limit=4`) | `IMPLEMENTED_UNVERIFIED` (no PG16) |
| Admission negative cases | 54 | 2 product orders × 27 cases in `TestAdmissionFailures` = **54** | `IMPLEMENTED_UNVERIFIED` (no Go) |
| Effect-boundary cases | 14 | 2 product orders × 7 modes in `TestEffectBoundaries` = **14** | `IMPLEMENTED_UNVERIFIED` (no Go) |
| Crash matrix subcases | 12 (CR-01..CR-10) | 12 `cases` entries in `TestCrashMatrix` + `TestCommittedDisconnect` (CR-04) | `IMPLEMENTED_UNVERIFIED` (no Go) |

**Test Quality Observations:** Assertions inspect actual SQLite row counts, execution IDs, signed receipt bytes, and cross-table invariants without silent error suppression. Two test-design limitations were identified: (1) `CR09-revoked-grant` only tests `site_authoriser` revocation, missing `ops_job` revocation (`AUD-S2-P1-01`); (2) `TestConcurrentDuplicates`/`TestCompetingOperations` are serialized by the synchronous `AcceptUnix` loop (`AUD-S2-P2-01`).

---

## 13. Findings Register

See machine-readable register in `O1-Sprint-2-Findings.json`. All 8 technical findings (`0 P0, 2 P1, 3 P2, 3 P3`) and 6 governance blockers (`GOV`) are summarized below:

| ID | Severity | File & Line | Summary | Blocks Sprint 3? | Blocks Prod? |
|---|---|---|---|---|---|
| `AUD-S2-P1-01` | **P1** | `services/site-agent/agent.go:159`, `services/local-helper/deployment.go:46` | Revoking `ops_job` key wedges pending deployment reconciliation/fencing (CR-09 gap) | **Yes** | **Yes** |
| `AUD-S2-P1-02` | **P1** | Audit environment / `.github/workflows/foundation.yml` | Go, Linux process/CR-01..CR-10, TS, PG16 RLS, and hosted CI `45594545137` unverified at runtime in this sandbox | **Yes** | **Yes** |
| `AUD-S2-P2-01` | **P2** | `services/local-helper/socket_linux.go:33`, `deployment.go:150-227` | Serial socket accept loop masks multi-connection concurrency & causes 4s head-of-line blocking | No | Yes |
| `AUD-S2-P2-02` | **P2** | `services/site-agent/deployment.go:35-72`, `services/local-helper/deployment.go:65-150` | SQLite `BEGIN IMMEDIATE` write lock held across live Unix-socket HTTP calls | No | Yes |
| `AUD-S2-P2-03` | **P2** | `packages/local-policy/policy.go:156-167` | Hardened `MC-14` `BackupForJob` checks skipped when `c.LocalCheck == nil` | No | Yes |
| `AUD-S2-P3-01` | **P3** | `README.md:9,58`, `O1-Sprint-2.md:8,195,221`, `O1-Sprint-2-Checkpoint.md:27` | Stale documentation claims Sprint 2 planned/uncommitted at `8f9d88be...`; BRD v1.2 footers show `v1.1` | No | No |
| `AUD-S2-P3-02` | **P3** | `services/site-agent/deployment.go:165-168` | `job_stage.state` records `'FAILED'` instead of `'UNKNOWN'` for `UNKNOWN` outcomes | No | No |
| `AUD-S2-P3-03` | **P3** | `tests/sprint2/central.go:69,148-155` | Central simulator hardcodes single test enrolment (`fixture.Install`, `fixture.AgentID`, `mock-receipt`) | No | No |
| `AUD-S2-GOV-01..06` | **GOV** | `docs/adr/OD01..OD08`, `services/ops-api/app.ts`, `products/simulator/` | Unsigned `OD01–OD08`/`Q01–Q07`, joint `LOCAL-01` sign-off, OS service accounts, TPM/PKI/TUF/mTLS, OIDC IAM, physical power-loss/OCI/clinical gates | No (test-only) | **Yes** |

---

## 14. Historical O0 Finding Closure Assessment

No previously closed O0 finding was reopened. Based on `docs/implementation/finding-lineage.json` and the O0 v1.2 re-audit HTML:
- **Verified Closed (Design & Fixture Layer):** `AUD-P0-01` (envelope scope separation), `AUD-P0-03`/`NEW-P0-01` (MC-09 site-authoriser tenant+site+target binding — independently re-verified at runtime), `AUD-P0-04` (typed MC-03 params), `AUD-P0-06` (production RFC 8785 JCS in Go/TS — source-verified, corpus runtime-verified in Node), `NEW-P1-01` (wrapped signed `AUX-01`), `NEW-P2-01..03` (MC-10 128-char ref, LOCAL-01 root `additionalProperties: false`, reading-copy hygiene).
- **Implemented in Test-Only Simulation (Awaiting Production Gates):** `AUD-P0-05`/`07` (helper isolation & bilateral `SO_PEERCRED` — simulated under shared UID `65534`), `AUD-P0-08` (SQLite WAL durability, replay, monotonic clock watermark), `AUD-P0-09` (PHI-free schemas & ledgers), `AUD-P0-10` (PG16 composite FKs + FORCE RLS — pending live PG16 runtime verification), `AUD-P1-01`/`04` (two-product neutrality), `AUD-P1-08` (`LOCAL-01` locks/readiness — pending joint sign-off), `AUD-P1-09` (`MC-14` backup gate).
- **Open Governance / Production Gates:** `AUD-P0-11` (`OD01–OD08` & `Q01–Q07` — 0/8 signed), `AUD-P0-02` (real enrolment PoP/TPM), `AUD-P1-05` (real PKI/cert rotation), `AUD-P1-06` (OIDC/MFA/RBAC & TUF), `AUD-P1-07` (WORM audit anchoring).

---

## 15. Production Governance Blockers

1. **OD01–OD08 Human Sign-Off (`AUD-S2-GOV-01`):** All 8 ADRs in `docs/adr/OD01_FOR_APPROVAL.md`–`OD08_FOR_APPROVAL.md` have blank signature tables (`0/8` approved) and unresolved `Q01–Q07` policy/vendor selections.
2. **Joint `LOCAL-01` Approval (`AUD-S2-GOV-02`):** Operations + ZypoSono Clinical/Security sign-off (`ZypoSono_One_Master_BRD_v1.5.pdf` §22.2) remains open.
3. **OS Service-Account Isolation (`AUD-S2-GOV-03`):** Separate Linux users/groups, systemd hardening, and rootless OCI/Podman execution are not yet implemented.
4. **Production Trust & Custody (`AUD-S2-GOV-04`):** Hardware TPM 2.0, production CA/KMS, TUF anti-rollback metadata, and outbound mTLS remain gated.
5. **Operator IAM & Central API Ingress (`AUD-S2-GOV-05`):** `services/ops-api/app.ts` has no authenticated HTTP ingress, OIDC IdP, MFA, or RBAC middleware yet.
6. **Physical Power-Loss, Real OCI Execution, Hold Release & Clinical Commissioning (`AUD-S2-GOV-06`):** Separately gated for later O1/O2 phases.

---

## 16. Exact Independently Executed Test Evidence

All independent commands executed inside Linux `6.12.105+ x86_64` sandbox (`Node v18.20.4`, `Python 3`, isolated temp copy of `8f9d88be2013c9b014202fc2435ec1368d83575c`):

| Independent Check | Command / Harness | Exit Code | Observed Result |
|---|---|---:|---|
| Contract binding drift | `node scripts/generate-types.mjs` (in temp copy) + `diff -u` against `types.ts` & `types_generated.go` | 0 | `TS_BINDINGS_DRIFT: ZERO`, `GO_BINDINGS_DRIFT: ZERO` |
| Golden signed vectors (Ed25519 + ES256 + nested MC-09/MC-12) | Independent Node `node:crypto` verifier over `signed_vectors.json` & `examples/*.example.json` | 0 | `GOLDEN_SIGNED_VECTORS: 15/15 PASS` (exact `input_b64` match) |
| Negative signed vectors | Independent Node `node:crypto` verifier over `negative_signed_vectors.json` | 0 | `NEGATIVE_SIGNED_VECTORS: 2/2 REJECTED` |
| Adversarial cross-site MC-09 re-sign probe | Fresh Ed25519 `ops_approval` key re-signs `MC-09` with changed `site_id` + cloned `site_authoriser` key | 0 | `ADVERSARIAL_CROSS_SITE_MC09_RESIGN_REJECTED: true` (`BAD_SIGNATURE`) |
| Agent SQLite WAL/FK/uniqueness probes | Python `sqlite3` executing `agent_sqlite_v1_2.sql` + 6 adversarial constraint violation inserts | 0 | `AGENT_SQLITE_INVARIANT_PROBES: 6/6 REJECTED (WAL=wal, FK=1)` |
| Specification SHA-256 verification | Python `hashlib.sha256` against `docs/implementation/source-sha256.txt` | 0 | 5/5 `docs/` source files `MATCH`; 3/3 `docs/implementation/specifications/` reading copies differ as documented |
| Go / TS / PG16 / Hosted CI suites | Unavailable in audit sandbox (no `go`, no `postgres`/`psql`, no `node_modules`, private GitHub CI) | N/A | Recorded honestly as `IMPLEMENTED_UNVERIFIED` (`AUD-S2-P1-02`) |

---

## 17. Prioritised Remediation Requirements

1. **P1 (Required before Sprint 3):** Fix `AUD-S2-P1-01` in `services/site-agent/agent.go:159` and `services/local-helper/deployment.go:46` so already-admitted `pending_deployment` / `deployment_execution` rows can reconcile committed effects or fence unapplied reservations even after `ops_job` key revocation; add `CR09-revoked-ops-job-committed` and `CR09-revoked-ops-job-reserved` to `TestCrashMatrix`.
2. **P1 (Required before Sprint 3):** Close `AUD-S2-P1-02` by executing the consolidated Go (ordinary + `-race`), Linux `SO_PEERCRED`/crash-matrix, TypeScript, and PostgreSQL 16 integration suite on an equipped Linux runner (or providing signed hosted CI run `45594545137` logs bound to `8f9d88be2013c9b014202fc2435ec1368d83575c`) for independent verification.
3. **P2 (Recommended early Sprint 3):** Address `AUD-S2-P2-01` (bounded concurrent helper connections with per-job serialization around `reconcile()`), `AUD-S2-P2-02` (move `LOCAL-01` HTTP I/O outside SQLite `BEGIN IMMEDIATE` transactions), and `AUD-S2-P2-03` (enforce `BackupForJob` checks unconditionally when `BackupEvidenceRequired` is true).
4. **P3 (Housekeeping):** Update `README.md` and Sprint 2 reports (`AUD-S2-P3-01`), map `UNKNOWN` to `job_stage.state='UNKNOWN'` (`AUD-S2-P3-02`), and replace hardcoded single-site constants in `tests/sprint2/central.go` with an enrolment table in Sprint 3 (`AUD-S2-P3-03`).

---

## 18. Sprint 3 Entry Criteria & Final Scoring

### Domain Scores (Evidence-Based, Out of 10)

| Audit Domain | Score (/10) | Basis |
|---|---:|---|
| Architecture and product neutrality | 9.0 | Clean control/clinical separation, genuine two-product reuse, zero ZypoSono leakage |
| Contracts and cryptography | 9.0 | 15/15 golden + 2/2 negative + cross-site MC-09 verified at runtime; zero binding drift |
| Tenant isolation | 9.0 | Strong PG16 composite FKs + FORCE RLS + non-owner role + pooled TX wrapper (source-verified) |
| Agent durability | 8.5 | Strong SQLite WAL/FK/sequence/clock-watermark design (6/6 SQLite probes); minor P2 I/O-in-TX |
| Helper authorisation | 8.0 | Bilateral `SO_PEERCRED`, independent trust/re-verification, `O_NOFOLLOW`/`Flock`; shared UID limitation |
| Local maintenance safety | 9.0 | Strict `LOCAL-01` schema validation + atomic product-side `BEGIN IMMEDIATE` final safety check |
| Backup evidence | 8.5 | Independent `BackupForJob` policy on Agent & Helper; P2 `LocalCheck != nil` gate |
| Crash consistency | 7.5 | Strong CR-01..CR-10 design; capped by `AUD-S2-P1-01` (`ops_job` revocation wedge) & sandbox unverified |
| Idempotency and concurrency | 7.5 | Cross-table effect/no_effect fencing; capped by `AUD-S2-P2-01` serial accept loop |
| Receipt reconciliation | 8.0 | Pinned deterministic execution ID, immutable terminal receipts; single-site fixture constants |
| Privacy | 9.5 | Zero PHI, zero clinical DB access, zero production credentials, explicit test-only keys |
| Test quality | 8.0 | Exact count reconciliation (58 Win / 179 Linux / 31 TS / 5 PG16), strong DB assertions; CR-09 & serial gaps |
| Maintainability | 8.5 | Clean Go/TS structure, bounded wire I/O; minor stale docs (`README.md`, `O1-Sprint-2.md`) |
| Governance discipline | 9.5 | Honest 0/8 ADR status, zero silent approvals, explicit production NO_GO maintained |
| **Overall Engineering Score** | **8.5 / 10** | Strong test-only foundation; conditional on 2 blocking P1 items before Sprint 3 |

### Structured Audit Verdict Block

```text
AUDITED_COMMIT: 8f9d88be2013c9b014202fc2435ec1368d83575c
AUDIT_SCOPE: O1_SPRINT_1_AND_SPRINT_2_TEST_ONLY_INTEGRATED_AUDIT
O1_SPRINT_1_REGRESSION: PASS_SOURCE_AND_CRYPTO_SQLITE_PROBES_GO_PG16_UNVERIFIED_IN_SANDBOX
O1_SPRINT_2_INDEPENDENT_VERDICT: GO_WITH_CONDITIONS_TEST_ONLY
WAVE_2A_SOCKET_INTEGRATION: IMPLEMENTED_UNVERIFIED_WITH_P2_SERIAL_ACCEPT
WAVE_2B_PRODUCT_NEUTRALITY: VERIFIED_SOURCE_STRONG
WAVE_2C_LOCAL_MAINTENANCE: VERIFIED_SOURCE_ATOMIC_NO_TOCTOU
WAVE_2D_BACKUP_EVIDENCE: PARTIALLY_VERIFIED_WITH_P2_LEGACY_GATE
WAVE_2E_DISCONNECT_RECONCILIATION: IMPLEMENTED_UNVERIFIED_SOURCE_SOUND
WAVE_2F_FAILURE_RECOVERY: IMPLEMENTED_UNVERIFIED_SOURCE_SOUND
WAVE_2G_CRASH_CONSISTENCY: PARTIAL_P1_OPS_JOB_REVOCATION_WEDGE
WAVE_2H_SECURITY_HARDENING: VERIFIED_SOURCE_AND_CRYPTO_PROBES
WAVE_2I_MAINTAINABILITY: VERIFIED_SOURCE_WITH_P3_STALE_DOCS
CR01_TO_CR10_INDEPENDENTLY_VERIFIED: NO_SANDBOX_NO_GO_AND_CR09_OPS_JOB_GAP
DUPLICATE_EXECUTION_PREVENTION: VERIFIED_SOURCE_AND_SQLITE_CONSTRAINTS
APPROVAL_CONSUMPTION_INTEGRITY: VERIFIED_SOURCE_AND_SQLITE_CONSTRAINTS
CENTRAL_RECEIPT_BINDING: VERIFIED_SOURCE_SINGLE_SITE_HARNESS
UNRESOLVED_DEPLOYMENT_PROTECTION: VERIFIED_SOURCE_SITE_WIDE_HOLD
POSTGRESQL_TENANT_ISOLATION: VERIFIED_SOURCE_PG16_RUNTIME_UNVERIFIED_IN_SANDBOX
LINUX_PROCESS_RECOVERY: IMPLEMENTED_UNVERIFIED_NO_GO_IN_SANDBOX
GO_RACE_TESTS: IMPLEMENTED_UNVERIFIED_NO_GO_IN_SANDBOX
HOSTED_CI_CURRENT_COMMIT: UNVERIFIED_PRIVATE_REPO_45594545137
TEST_INTEGRITY: VERIFIED_SOURCE_COUNTS_RECONCILED_ZERO_INFLATION
P0_FINDINGS: 0
P1_FINDINGS: 2
P2_FINDINGS: 3
P3_FINDINGS: 3
GOVERNANCE_BLOCKERS: 6 (OD01_TO_OD08; Q01_TO_Q07; JOINT_LOCAL_01_SIGN_OFF; OS_SERVICE_ACCOUNTS; TPM_PKI_TUF_MTLS; OIDC_IAM_AND_CLINICAL_GATES)
OVERALL_ENGINEERING_SCORE: 8.5/10
OD01_TO_OD08_APPROVED: 0_OF_8
O0_PRODUCTION_GATE: NO_GO
PRODUCTION_TRUST_INTEGRATION_ALLOWED: NO
PRODUCTION_SERVICE_ACCOUNT_ISOLATION_VERIFIED: NO
O1_RUNTIME_VERIFIED: NO_FORMAL_GATE_PASS
CLINICAL_PRODUCTION_READY: NO_GO
REMEDIATION_REQUIRED: YES_2_P1_CONDITIONS_BEFORE_SPRINT_3
READY_FOR_O1_SPRINT_3_TEST_ONLY: CONDITIONAL_ON_AUD_S2_P1_01_AND_AUD_S2_P1_02
```
