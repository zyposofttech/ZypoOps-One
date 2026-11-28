# O1 Sprint 3 checkpoint

26 November 2026. Clean main at 92b3b4059dc96ea4ec78222d6774f4ad16873c23. Accepted source fingerprint 3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6. Rechecked all 42 baseline evidence files. Hosted foundation runs 45594545137, 45768586579 and 45768899686 succeeded. The independent reviewer accepted all eight technical findings for test-only entry. Acceptance report SHA-256 bb92034aaae3a229fb9cc8fd9b628ff0a6e79178d029ec9bec55208ca03d3e57. Six production governance blockers remain open.

The baseline has 33 Prisma models, transaction-local tenant and site context, FORCE RLS, signed contracts, Go execution and a SQLite central simulator. NestJS exposes the database provider. Central application services, signed wire persistence and an authenticated synthetic integration are Sprint 3 work. Next action is to implement scoped internal application services and test them against a disposable PostgreSQL cluster.

## 3A–3B

Implemented synthetic principal authentication, scoped registry, separate optional non-owner provisioning login and tenant database cleanup. New registry test passed for two tenants, three sites and four installations alongside all five previous PG groups. Changed services/ops-api/modules, packages/database and the disposable harness. Next action is enrolment, catalogue, orchestration and the signed evidence path.

## 3C–3F

Implemented MC-02 CSR/PoP and one-time HMAC claim consumption, synthetic MC-01 issuance, enrolled signer associations, signed MC-05/08/09 ingestion, installation sequences, approval consumption, dispatch, AUX-01, MC-04, MC-07 and MC-14. Files are in services/ops-api/modules, cmd/synthetic-csr, tests/central and migration 003. Type checking and lint passed. PostgreSQL exercised malformed proofs, expired claims and concurrent single-use consumption. Further integration checks are in progress.

## 3G–3I

Added scoped deployment plans, prerequisite assessment, staged configuration, history-chain and onboarding queries. An inherited private pipe connects the existing Agent and Helper to the Nest services. The Linux worker runs only the two existing simulator profiles as an unprivileged user. Next action is to verify the complete topology and restart boundaries, then run consolidated regressions.

## Final checkpoint · 27 November 2026

The final candidate is `317b3a84a38c49e3f6a94cb6a4415fc7afb4f4e23e8fbf3a30458b0044d8dc29` on accepted baseline `92b3b4059dc96ea4ec78222d6774f4ad16873c23`. The Linux race rebuild and all five installed test packages completed under UID/GID 65534. Windows ordinary and race each passed 86 named outcomes. Linux ordinary and race each passed 228 with one standalone child entry skip. TypeScript contracts passed 31/31, interoperability 2/2, PostgreSQL tenancy 5/5 and central management 13/13. The final central regression included actual child termination on both sides of terminal receipt commit. Four installations each recorded a single simulated effect after lost acknowledgement and original receipt redelivery.

Changed paths include services/ops-api/modules, cmd/synthetic-csr, packages/database, tests/central, the disposable development harness, package scripts, foundation CI, the README milestones and Sprint 3 documentation. The accepted Go Agent/Helper and previous tests are preserved. All 30 raw logs verified against their checksums. Candidate CI and independent Sprint 3 review remain outstanding. Next action is independent review of the exact candidate in the [implementation report](O1-Sprint-3.md), [service catalogue](O1-Sprint-3-API-Service-Catalogue.md), [traceability register](O1-Sprint-3-Requirement-Traceability.json) and [runtime evidence](../audit/O1-Sprint-3-Runtime-Evidence.md).
