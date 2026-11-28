# O1 Sprint 3 runtime evidence

27 November 2026. Uncommitted candidate on baseline `92b3b4059dc96ea4ec78222d6774f4ad16873c23`. Source SHA-256 `317b3a84a38c49e3f6a94cb6a4415fc7afb4f4e23e8fbf3a30458b0044d8dc29`. All **30 raw logs** in the register passed byte-level SHA-256 verification. The register includes argument arrays, available environment overrides, exit codes and elapsed durations. Named Go and Node outcomes remain in the raw outputs. The evidence is unsigned. [Machine-readable register](sprint-3-evidence/index.json)

## Final local regression

| Command or harness | Result | Elapsed |
|---|---|---:|
| `pnpm lint` and `pnpm typecheck` | Both exited 0 | 2.11 s / 3.296 s |
| `pnpm test` | 31 contract tests passed, zero failures or skips | 0.766 s |
| `pnpm test:interop` | 2 passed, ES256 and Ed25519 across TypeScript and Go | 1.141 s |
| `node infrastructure/local-development/tenancy.mjs` | 5/5 PostgreSQL groups plus 13/13 central tests. The central count comprises two top-level tests and eleven subtests. Zero failures or skips. | 29.015 s |
| `go vet ./...` for Windows and Linux | Both exited 0 | 18.937 s / 14.344 s |
| Scratch contract generation and comparison | Generated Go and TypeScript targets matched tracked files after LF normalisation | 0.062 s / 0.11 s |
| `gofmt -l` on new Go sources and `git diff --check` | Both exited 0, no formatting or whitespace findings | 0.016 s / 0.046 s |

The database harness started an isolated PostgreSQL 16.13 instance, applied 001_o0, 002_runtime and 003_central, ran migration deploy a second time, and generated Prisma Client 6.19.3. A non-owner runtime login ran the 120-transaction, four-connection NestJS/Prisma RLS regression. Separately scoped principals and a non-owner provisioning login completed the central business-operation tests. The cluster stopped at the end. [PostgreSQL and central outputs](sprint-3-evidence/postgres-central.log)

### Go named outcomes

| Platform and package | Parents passed | Subtests passed | Skipped | Failed | Seconds |
|---|---:|---:|---:|---:|---:|
| [go-windows](sprint-3-evidence/go-windows.log) | 18 | 68 | 0 | 0 | 11.5 |
| [race-windows](sprint-3-evidence/race-windows.log) | 18 | 68 | 0 | 0 | 28.141 |
| [linux-cryptography](sprint-3-evidence/linux-cryptography.log) | 5 | 24 | 0 | 0 | 5.406 |
| [linux-local-helper](sprint-3-evidence/linux-local-helper.log) | 3 | 16 | 0 | 0 | 1.313 |
| [linux-local-policy](sprint-3-evidence/linux-local-policy.log) | 1 | 26 | 0 | 0 | 0.672 |
| [linux-site-agent](sprint-3-evidence/linux-site-agent.log) | 8 | 5 | 0 | 0 | 6.281 |
| [linux-sprint2](sprint-3-evidence/linux-sprint2.log) | 19 | 121 | 1 | 0 | 40.031 |
| [linux-race-cryptography](sprint-3-evidence/linux-race-cryptography.log) | 5 | 24 | 0 | 0 | 1.735 |
| [linux-race-local-helper](sprint-3-evidence/linux-race-local-helper.log) | 3 | 16 | 0 | 0 | 3 |
| [linux-race-local-policy](sprint-3-evidence/linux-race-local-policy.log) | 1 | 26 | 0 | 0 | 3.359 |
| [linux-race-site-agent](sprint-3-evidence/linux-race-site-agent.log) | 8 | 5 | 0 | 0 | 15.078 |
| [linux-race-sprint2](sprint-3-evidence/linux-race-sprint2.log) | 19 | 121 | 1 | 0 | 108.594 |

Windows recorded 86 pass events per run. Linux recorded 228 per run, with one standalone `TestServiceProcess` entry skipped. The parent harness invokes that entry as real subprocesses. Race repeats the same coverage. The Linux Sprint 2 package includes 21 crash-matrix subcases and separate committed-disconnect coverage. Cross-compilation alone was excluded from these runtime counts.

## Recovery and isolation evidence

| Witness | Observable outcome |
|---|---|
| Scoped registry | Two legal customers, three sites, four installations. Sibling and foreign scope mutations denied. Forged principal rejected. |
| Bootstrap races | One consumer won each concurrent pair. Expired, malformed, wrong-scope, wrong-key and reused requests left the tested claim state intact or already consumed. |
| Consent and backup | Correctly scoped dual signatures accepted. Independent authoriser identity and expected backup manifest were pinned before admission. The site held a competing product and its unconsumed grant. |
| Two-product transport | Each enrolled Linux worker used a private inherited pipe for central traffic and Unix peer-checked sockets locally. Each installation recorded one effect, one Agent consumption, one Helper consumption and one acknowledged receipt. |
| Terminal commit | The test killed a dedicated NestJS process at the receipt transaction callback before commit and killed a second after commit. A different application context observed absence then the original receipt. A subsequent commit/acknowledgement disconnect produced byte-identical redelivery. |
| Expiry and revocation | An undelivered expired job retained its consumed grant. A delivered expired job required reconciliation. Key revocation during the last installation's redelivery preserved the committed receipt. |
| Governance projections | Entitlement expiry preserved original deployment evidence. Configuration staging returned the LOCAL-01 application gate. The technical onboarding projection kept clinical and production at NO_GO. |

The central test names, durations and failure counts appear in the [integration log](sprint-3-evidence/postgres-central.log). The pipe harness and dedicated child implement the receipt commit barriers. [Test driver](../../tests/central/pipe.ts) · [Child process](../../tests/central/receipt-process.ts)

## Reproduce

Use Node 24.14.0, pnpm 10.32.1, Go 1.27.4, PostgreSQL 16 and an unprivileged user. On a native Linux host with PostgreSQL 16 binaries in `PGBIN` and Go on PATH:

```sh
pnpm install --frozen-lockfile
pnpm db:generate
pnpm generate
git diff --exit-code -- packages/management-contracts
pnpm lint
pnpm typecheck
pnpm test
pnpm test:interop
PGBIN=/usr/lib/postgresql/16/bin pnpm test:central
go vet ./...
go test -count=1 -v ./...
go test -race -count=1 -v ./...
```

The local run used Windows x64, WSL2 Linux 6.6.87.2 and UID/GID 65534. On Windows, provide Go and PostgreSQL binaries on PATH or set `PGBIN`. The central harness builds a Windows CSR utility and a Linux worker, then runs the worker through WSL as nobody. The local ordinary Linux suites used GOOS=linux, GOARCH=amd64 and CGO_ENABLED=0. Race binaries used CGO, Zig and a workspace glibc loader, then executed inside WSL. Windows race used clang. [Tool versions](sprint-3-evidence/tool-versions.log) · [Linux kernel](sprint-3-evidence/linux-platform.log) · [UID and GID](sprint-3-evidence/linux-identity.log) · [Exact commands](sprint-3-evidence/index.json)

To check this working tree, run `python scripts/verify-sprint2r-evidence.py --fingerprint` and compare the output to the register's `source_sha256`. That existing fingerprint algorithm includes Git-visible paths outside documentation and excludes README and .gitignore. Verify each registered raw file as bytes:

```python
from pathlib import Path
import hashlib, json
root = Path("docs/audit/sprint-3-evidence")
register = json.loads((root / "index.json").read_text(encoding="utf-8"))
for run in register["runs"]:
    assert run["exit_code"] == 0
    assert hashlib.sha256((root / run["log"]).read_bytes()).hexdigest() == run["sha256"]
```

The three successful hosted foundation runs apply to the accepted lineage. Candidate CI has not run because this working tree remains uncommitted. The hosted workflow is wired to run the new central suite on publication. [Hosted run metadata](sprint-3-evidence/hosted-baseline.log)
