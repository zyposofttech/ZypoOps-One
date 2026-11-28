# ZypoOps-One

**A product-neutral operations and deployment control plane for Zyposoft software.**

ZypoOps One is being developed to manage the lifecycle of Zyposoft applications installed at customer locations: site registration, secure agent identity, software release delivery, operational health, authorised maintenance, and auditable execution.

The platform separates **central operations management** from **customer-local application data and clinical control**. It is designed to support ZypoSono One initially and additional Zyposoft products through the same management protocols, without embedding product-specific clinical logic in the Site Agent.

> **Project status:** Active development. O1 Sprints 1 and 2 are committed for synthetic engineering verification. Sprint 2R received independent acceptance for synthetic engineering on 26 November 2026. The hosted [O1 foundation CI workflow](https://github.com/zyposofttech/ZypoOps-One/actions/workflows/foundation.yml) has passed. The platform is **not approved for customer deployment, production-trust integration or clinical use**.

## What ZypoOps One is designed to do

- **Manage multiple products and sites:** A shared operational model for customer organisations, installations, agents and globally reusable signed software artifacts.
- **Verify management commands:** Contract-based messages with Ed25519/ES256 signatures, signing-key purpose restrictions, tenant/site binding and replay protection.
- **Support controlled updates:** Signed release references, site-authorised actions, local readiness checks, backup evidence and recoverable execution workflows.
- **Keep sites operational during disconnections:** A local Go agent with a durable SQLite WAL ledger and resilient receipt handling.
- **Preserve customer data boundaries:** Patient records, medical images and clinical decisions remain with the customer-local application; routine central PHI collection is outside the management model.
- **Provide accountable operations:** Tenant isolation, structured technical telemetry and traceable execution evidence.

These are platform goals. Features not explicitly listed as implemented below may still be design-stage or scheduled for a later sprint.

## Architecture

```text
                    CENTRAL OPERATIONS (planned services)
       +---------------------------------------------------+
       | Operations Console   | NestJS Operations API      |
       | PostgreSQL + Prisma  | Signed jobs / audit trail  |
       +----------------------+----------------------------+
                                      |
                         Outbound agent connection
                                      |
                    CUSTOMER SITE / ON-PREMISES
       +---------------------------------------------------+
       | Go Site Agent                                     |
       | - Contract and signature verification             |
       | - SQLite WAL job and receipt ledger               |
       | - Offline/restart recovery                        |
       +----------------------------+----------------------+
                                    | Restricted Unix socket
       +----------------------------v----------------------+
       | Local Helper (independent authorisation boundary) |
       | - Peer credential validation                      |
       | - Signature / consent / artifact re-verification  |
       | - Allow-listed operations                         |
       +----------------------------+----------------------+
                                    |
                   Product-specific local management API
                       +------------+-------------+
                       |                          |
                 ZypoSono One             Other Zyposoft apps
                 (local authority)         (local authority)
```

Sprint 2 connects the agent and unprivileged helper over a Unix socket and reconciles two simulated products. The Sprint 1 foundation contains a NestJS application-context foundation, contract and signing libraries, PostgreSQL isolation tests, a Go agent with a mocked outbound transport, and an unprivileged helper that records simulated effects. The complete console, production transport and real installation execution are **not implemented or approved**.

## Engineering progress

| Milestone | Status | Verified scope |
|---|---|---|
| O0 — Architecture and contract design | Conditional design approval | Technical baseline accepted for controlled development; governance decisions remain open |
| O1 Sprint 1A — Contracts and cryptography | Complete (test-only) | 15 signed management messages, cross-runtime signing and negative tests |
| O1 Sprint 1B — Database isolation | Complete (test-only) | PostgreSQL 16 migrations and tenant/site RLS tests |
| O1 Sprint 1C — Go Site Agent | Complete (test-only) | Durable jobs, signed receipts, replay protection and restart recovery |
| O1 Sprint 1D — Local Helper | Complete (test-only) | Real Linux peer-credential checks and independent authorisation tests |
| O1 Sprint 2 — End-to-end integration | Committed, test-only (25 November 2026) | Agent–helper socket integration, two-product simulation, local locks, signed backup evidence and receipt reconciliation |
| O1 Sprint 2R — Independent acceptance | GO_TEST_ONLY (26 November 2026) | 8/8 technical findings closed, 42/42 evidence files verified. [Acceptance report](docs/audit/O1-Sprint-2R-Independent-Acceptance.md) |
| Production deployment and clinical readiness | Not approved | Requires remaining governance approvals, runtime integration, independent testing and site commissioning |

The [Sprint 1 engineering report](docs/implementation/O1-Sprint-1.md) records commands, test results and scope limitations. See the [O0 v1.2 independent audit](docs/zyposoft_operations_platform_independent_o0_v1_2_architecture_security_re_audit.html) for the design-gate conditions.

## Technology stack

| Layer | Technologies |
|---|---|
| Backend foundation | NestJS 11, TypeScript, Node.js 24 |
| Database | PostgreSQL 16, Prisma 6, row-level security |
| Site-side services | Go, SQLite WAL, Linux Unix-domain sockets |
| Contracts | JSON Schema Draft 2020-12; MC-01–MC-14, AUX-01 and LOCAL-01 |
| Security foundation | RFC 8785 canonical JSON, Ed25519, ES256, key-purpose and site-scope enforcement |
| Tooling and CI | pnpm 10, Go tests, Node test runner, GitHub Actions |

Exact versions are pinned in [`package.json`](package.json), [`.node-version`](.node-version), [`go.mod`](go.mod), `pnpm-lock.yaml` and `go.sum`.

## Repository layout

```text
.github/workflows/          Foundation CI
packages/
  database/                Prisma models, SQL migrations and tenant context
  management-contracts/    Schemas, generated types and strict decoding
  signing-ts/              TypeScript signing and verification
  signing-go/              Go signing and verification
  local-policy/            Site-side authorisation rules
  local-store/             SQLite access
services/
  ops-api/                 NestJS application-context foundation
  site-agent/              Go Site Agent and recovery tests
  local-helper/            Restricted helper and Linux socket tests
infrastructure/
  local-development/       Disposable PostgreSQL integration harness
tests/                       Contracts, cryptography, tenancy and fixtures
docs/
  adr/                     Proposed architecture decision records
  implementation/          Sprint reports, source lineage and specifications
```

## Local development

### Prerequisites

- Node.js and pnpm versions specified in the repository.
- Go toolchain version specified in `go.mod` (required for interoperability and Go tests).
- PostgreSQL **16** binaries (`initdb`, `pg_ctl`, `postgres`, `psql`) for the isolated database tests.
- A Linux environment for `SO_PEERCRED` socket tests (Ubuntu or suitable WSL2 environment).

On Windows, use PowerShell in the repository root.

```powershell
 git clone https://github.com/zyposofttech/ZypoOps-One.git
 cd ZypoOps-One
 corepack enable
 pnpm install --frozen-lockfile
 pnpm generate
 pnpm db:generate
```

### Run foundation checks

```powershell
pnpm lint
pnpm typecheck
pnpm test
pnpm test:interop
go vet ./...
go test -count=1 ./...
```

For Go race detection, use a supported CGo compiler/toolchain:

```powershell
go test -race -count=1 ./...
```

### Test PostgreSQL tenant isolation

The repository includes a **disposable, loopback-only PostgreSQL 16 test harness**. It creates a temporary test cluster, applies migrations, executes isolation tests and removes the cluster when finished. **Do not point this harness at an existing or production database.**

```powershell
$env:PGBIN = 'C:\Program Files\PostgreSQL\16\bin'
node infrastructure/local-development/tenancy.mjs
```

On Linux, set `PGBIN` to the installed PostgreSQL 16 binaries directory and run the same script.

## Verification evidence

Sprint 1 reported the following locally executed results:

- **31 TypeScript contract tests** passed, plus bidirectional Ed25519 and ES256 Go/TypeScript interoperability tests.
- **5 PostgreSQL test groups** passed, including **120 concurrent NestJS/Prisma operations** through a four-connection pool with no observed tenant or site leakage.
- **57 Windows Go test/subtest pass events** and **32 Linux agent/helper pass events**; the coverage overlaps and should not be added as distinct tests.
- **100 simulated agent crash iterations**, plus child-process termination and recovery checks.
- Linux kernel-supplied Unix socket peer-credential and helper policy checks passed.
- Lint, type checking, Go vet, Go race detection and generated-file drift checks passed.
- The hosted **[O1 foundation workflow](https://github.com/zyposofttech/ZypoOps-One/actions/workflows/foundation.yml)** passed for the committed Sprint 1 foundation.

These are **engineering test results, not security certification**. Hardware power loss, TPM-backed key custody, real customer deployment, full production CA/KMS integration and clinical commissioning remain outside Sprint 1 acceptance.

## Security, privacy and operational boundaries

- The local clinical application remains authoritative for its own clinical data and workflows.
- The management control plane must not become a routine repository for patient records or DICOM studies.
- Agent and helper permissions must remain separated; signed central instructions alone are not sufficient for high-risk local actions.
- Test signing keys and synthetic data in this repository are **not production credentials**.
- Proposed architecture decisions **OD01–OD08**, Q01–Q07 policy choices and joint local-interface sign-off are still awaiting formal approval.
- No real hospital installation, production privileged-helper deployment, production-trust integration or clinical use is authorised by this repository's current status.

Please do not report vulnerabilities using public issues that disclose exploit details or secrets. Contact the maintainers privately through the project organisation.

## Documentation

- [O1 Sprint 1 implementation and test report](docs/implementation/O1-Sprint-1.md)
- [Architecture decisions awaiting approval](docs/adr/)
- [Operations Platform Master BRD v1.2](docs/implementation/specifications/Zyposoft_Operations_Platform_Master_BRD_v1.2.pdf)
- [O0 Consolidated Technical Design v1.2](docs/implementation/specifications/Zyposoft_Operations_Platform_O0_Consolidated_Remediation_v1.2.pdf)
- [ZypoSono One Master BRD v1.5](docs/implementation/specifications/ZypoSono_One_Master_BRD_v1.5.pdf)

## Project ownership

Developed by **Zyposoft Technologies Private Limited**.

The repository is publicly visible, but **no licence file is currently published**. Do not assume that public visibility grants permission to reuse or redistribute the source. For collaboration, licensing or deployment enquiries, contact Zyposoft through its official channels.
