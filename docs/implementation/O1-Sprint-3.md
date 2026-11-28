# O1 Sprint 3 implementation report

27 November 2026 · Test-only engineering candidate on `main`

**Ready for independent Sprint 3 audit.** The PostgreSQL-backed NestJS application dispatched signed jobs to four enrolled installations across two tenants and three sites. The existing Go Agent, restricted Local Helper and both product profiles processed them under Linux. Each installation recorded one simulated effect, one consumption in each local approval ledger, one acknowledged Agent receipt and one accepted central receipt. The candidate remains local and uncommitted.

## Accepted entry and evidence

Work began on a clean tree at `92b3b4059dc96ea4ec78222d6774f4ad16873c23`. The independently accepted Sprint 2R runtime fingerprint was `3efaa1b6ae9045d0b10b978e1103af4bcd6182e8aedde4f258859b6b61c4f9c6`. The reviewer closed 8/8 technical findings with GO_TEST_ONLY and a score of 9.2/10. Entry verification reproduced all 42 evidence hashes and checked the three successful hosted runs. The archived original acceptance report has SHA-256 `bb92034aaae3a229fb9cc8fd9b628ff0a6e79178d029ec9bec55208ca03d3e57`. [Independent acceptance](../audit/O1-Sprint-2R-Independent-Acceptance.md) · [Hosted run metadata](../audit/sprint-3-evidence/hosted-baseline.log)

The Sprint 3 candidate source SHA-256 is `317b3a84a38c49e3f6a94cb6a4415fc7afb4f4e23e8fbf3a30458b0044d8dc29`. The README milestone register records the independent entry decision. The prior SQLite central simulator, Go Agent, Helper, product-local protocol, signed wire contracts and previous test sources retain their accepted implementation. [Runtime evidence](../audit/O1-Sprint-3-Runtime-Evidence.md)

## Waves and BRD alignment

| Wave | Working implementation | Verification |
|---|---|---|
| 3A · OP01/36 | NestJS providers use a scoped principal issued by the current synthetic context, a bounded error catalogue and transaction-local Prisma scope. Both pools close with the app. | Forged context rejected. Application reopens and reauthenticates during dispatch and receipt processing. |
| 3B · OP01/02/33 | Customer, contracting legal entity, site, host binding and installation have distinct identities. Replacement records reference the predecessor. Lifecycle changes preserve evidence and ownership transfer has an explicit ceremony gate. | Two tenants, three sites and four installations. Pooled reads, cross-scope replacement attempts and retained installation history checked. |
| 3C · OP10/11 | Short-lived randomised claim, genuine Ed25519 PKCS#10, MC-02 PoP and HMAC, purpose-bound enrolment, signed synthetic MC-01 and revocation history. | Wrong proof, wrong CSR key, wrong installation/installer, expiry, duplicate enrolment and concurrent consumption exercised. |
| 3D · OP03/04/12/29 | Global signed artifact plus immutable catalogue metadata, scoped signed entitlement revisions and dual-signed approval grants issued against a prior request. | Identical artifacts reused by two installations per product. Conflicting metadata, wrong checker, expiry, signature substitution and changed target rejected. |
| 3E · OP12/36 | An installation-scoped sequence, signed MC-03 bundle, central grant consumption and outbox record commit together. Dispatch attempts, expiry and site holds survive reopen. | Concurrent duplicate creation returned one identity. Changed inputs conflicted. Separate installations used sequence one. Delivered and undelivered expiry produced distinct durable states. |
| 3F · AUX-01/MC-04/07/14 | Signed progress and terminal receipts, enrolled telemetry signer, bounded diagnostic codes, last-known and stale health, and plan-bound signed backup evidence. | Duplicate progress, wrong first receipt, conflicting terminal receipt, late progress, sibling signer, technical input allow-list and unencrypted backup cases exercised. |
| 3G · OP13/14/15 | Typed prerequisites and staged configuration use scoped approvals, expected revisions and synthetic secret references. Management retry is bounded. | Configuration revision and reuse conflicts, entitlement expiry, acknowledgement loss, Agent reopen and local simulation continuity checked. |
| 3H · OP29/33/36 | Correlated append-only events form per-site hash chains. A technical onboarding query derives readiness from durable records. | Denials recorded, history verified and technical onboarding reported for all four successful installations. Clinical and production status stayed NO_GO. |
| 3I · integrated management | A bounded test worker drives the accepted local stack through a credential-bound private pipe. Both profiles use the same central services and worker binary. | Four end-to-end runs, central restart during pending admission, actual process termination around terminal commit and original receipt redelivery. |

The module implementations and their input/output types, authorised callers, error semantics, retry rules and local application gates are documented in the [internal service catalogue](O1-Sprint-3-API-Service-Catalogue.md). The [requirement traceability register](O1-Sprint-3-Requirement-Traceability.json) maps each wave to its source and tests.

## Persistence and authority

Migration `003_central` extends the accepted 33-model database with one new scoped entity, `deployment_plan`. Its immutable association binds the intended installation and signed artifact to independently expected backup metadata and prerequisite results before backup evidence arrives. Existing models store customer/site metadata, installation lineage, scoped credentials and key history, signed manifests, entitlements and approvals, signed command and receipt wires, installation sequences, progress, health, configuration and audit correlations. The migration adds composite scoped foreign keys, unique constraints and FORCE RLS on the plan. Nullable additions preserve the older rows and migration files. [Migration](../../packages/database/migrations/003_central/migration.sql) · [Prisma mappings](../../packages/database/schema.prisma)

The disposable harness supplies an optional separate, non-owner provisioning login. It has named table grants for scoped provisioning, authoriser registration and global artifact publication. Business queries use the existing runtime login and transaction-local tenant/site wrapper. Neither application role has BYPASSRLS. Approval association and configuration staging consume grants inside the corresponding database transaction. Failed transactions roll back consumption. The database harness applied three migrations to a fresh PostgreSQL 16.13 cluster and found zero pending on replay. [Database evidence](../audit/sprint-3-evidence/postgres-central.log)

**Enrolment.** The operator issues a claim pinned to the installation and installer. The installer presents the contract-defined HMAC and a signed MC-02 containing a verified CSR. Claim consumption, MC-01 identity, synthetic serial, Agent and role-specific signing-key records commit together. Revocation retains history. The verifier implements the synthetic Ed25519 path. The serial identifies a test record. Production certificate issuance remains gated.

**Authorisation.** The central registry stores an artifact-scoped MC-05 by canonical digest. An MC-08 revision describes the synthetic site's entitlement. A maker requests an operation against an exact target. A different checker submits the MC-09 with independently generated site consent. Private site-authorisation material stays with the test authority. The Agent consumes the grant on its own local admission and the Helper consumes it on its independent reservation. Network retries reuse the original signed evidence.

**State and delivery.** Job preparation ends with RECEIVED → VALIDATED → ACCEPTED in the central transaction. ACCEPTED means eligible for scoped retrieval. Delivery attempts and AUX-01 progress have separate records. The first bound MC-04 projects its supported terminal state inside the receipt transaction, using RUNNING where required by the database transition rules. A report of REJECTED can project directly from ACCEPTED. Exact receipt redelivery returns successfully. An expired job with prior delivery retains its hold and reports RECONCILIATION_REQUIRED. An expired undelivered job can be rejected without releasing its grant.

## Recovery, limits and next decision

The harness reopened the central application before delivery, after persisting attempts, during progress and after the Agent had durably admitted its job. It lost the first acknowledgement after terminal commit. The Agent reopened its SQLite ledger, redelivered its original signed receipt and left the product effect count at one. A separate NestJS process was killed immediately before and after transaction commit. The parent independently observed an absent terminal row at the first boundary and the original receipt at the second. The last installation also redelivered after revocation of the original Ops job key. [Integration witnesses](../../tests/central/pipe.ts) · [Named outcomes](../audit/sprint-3-evidence/postgres-central.log)

The previous 21-case crash matrix and the separate CR04 regression passed in ordinary and race runs. The two generic products remain synthetic. In the new harness the local roles share a non-root OS identity, and central restarts use the same PostgreSQL instance. Power loss, separate service accounts, production IAM/PKI, physical media and clinical commissioning remain separate decisions. Configuration application requires joint LOCAL-01 approval. The audit chain has local integrity verification and awaits an approved external anchor.

All six governance findings AUD-S2-GOV-01 through -06 remain open. OD01–OD08 have zero human approvals and Q01–Q07 remain open. The next engineering step is independent review of this candidate fingerprint, including provisioning/RLS grants, signed input binding, central transaction boundaries and the four-installation run. A hosted run against the candidate needs separately authorised publication. Sprint 4 awaits independent Sprint 3 acceptance and its own scoped entry decision.

## Final verdict

```text
O1_SPRINT_3_STATUS: IMPLEMENTED_AND_VERIFIED_TEST_ONLY
ACCEPTED_BASELINE_COMMIT: 92b3b4059dc96ea4ec78222d6774f4ad16873c23
SPRINT_2R_INDEPENDENT_ACCEPTANCE_PRESERVED: YES_8_OF_8_FINDINGS_ACCEPTED
SPRINT_2R_REPORT_ARCHIVED: YES_ORIGINAL_BYTES_SHA256_VERIFIED
EXISTING_APPLICATION_SOURCE_PRESERVED: YES_AGENT_HELPER_SIMULATORS_AND_PREVIOUS_TESTS
WAVE_3A_CENTRAL_APPLICATION: PASS_INTERNAL_NESTJS
WAVE_3B_CUSTOMER_SITE_INSTALLATION: PASS_2_TENANTS_3_SITES_4_INSTALLATIONS
WAVE_3C_SYNTHETIC_ENROLMENT: PASS_ED25519_TEST_IDENTITY
WAVE_3D_ARTIFACT_ENTITLEMENT_APPROVAL: PASS_SIGNED_SYNTHETIC_LIFECYCLES
WAVE_3E_JOB_ORCHESTRATION: PASS_POSTGRESQL_DISPATCH
WAVE_3F_RECEIPTS_TELEMETRY_BACKUP: PASS_SCOPED_SIGNED_EVIDENCE
WAVE_3G_CONFIG_PREREQUISITES_OFFLINE: PASS_STAGING_AND_TEST_CONTINUITY
WAVE_3H_OPERATIONAL_AUDIT_ONBOARDING: PASS_LOCAL_CHAIN_AND_TECHNICAL_PROJECTION
WAVE_3I_INTEGRATED_VERTICAL_SLICE: PASS_FOUR_INSTALLATIONS
CENTRAL_NESTJS_APPLICATION: PASS
POSTGRESQL_BACKED_DISPATCH: PASS
TEST_ONLY_TRANSPORT_ISOLATION: PASS_AUTHENTICATED_INHERITED_PIPE
SYNTHETIC_ENROLMENT_POP: PASS_ED25519_CSR_AND_MC02
SINGLE_USE_ENROLMENT_CLAIMS: PASS_ONE_CONCURRENT_WINNER
AGENT_IDENTITY_BINDING: PASS
GLOBAL_ARTIFACT_REUSE: PASS_TWO_PRODUCTS_EACH_REUSED
ENTITLEMENT_CONTINUITY: PASS_EXPIRED_MANAGEMENT_STATE_LOCAL_EVIDENCE_RETAINED
INDEPENDENT_SITE_CONSENT: PASS_DUAL_PROOF
JOB_SEQUENCE_INTEGRITY: PASS_INSTALLATION_SCOPED
JOB_IDEMPOTENCY: PASS
AUX01_PROGRESS: PASS_TEST_TRANSPORT
MC04_RECEIPT_IMMUTABILITY: PASS
MC07_TELEMETRY: PASS_SIGNED_BOUNDED_INGESTION
MC14_BACKUP_EVIDENCE: PASS_OPERATION_AND_SIGNER_BOUND
CONFIGURATION_STAGING: PASS_LOCAL_APPLICATION_GATED
TECHNICAL_ONBOARDING: PASS_SYNTHETIC
OPERATIONAL_AUDIT: PASS_SCOPED_CHAIN_EXTERNAL_ANCHOR_GATED
TWO_PRODUCT_INTEGRATION: PASS
MULTI_TENANT_INTEGRATION: PASS
CENTRAL_RESTART_RECOVERY: PASS_CONTEXT_RESTART_AND_RECEIPT_PROCESS_TERMINATION
ACKNOWLEDGEMENT_LOSS_RECOVERY: PASS_IDENTICAL_RECEIPT_REDELIVERY
DUPLICATE_EFFECT_PREVENTION: PASS_ONE_EFFECT_PER_EXECUTION
REVOKED_KEY_RECOVERY_REGRESSION: PASS_SPRINT_2R_MATRIX_AND_CENTRAL_REDELIVERY
UNRESOLVED_DEPLOYMENT_HOLDS: PASS
PHI_FREE_MANAGEMENT_PLANE: TEST_ONLY_SCHEMA_AND_ALLOWLIST_CHECKS_PASSED
DATABASE_MIGRATIONS: PASS_3_APPLIED_ZERO_PENDING_ON_REPLAY
POSTGRESQL_RLS: PASS_120_TRANSACTIONS_POOL_4
TYPESCRIPT_TESTS: PASS_31_CONTRACT_13_CENTRAL_5_TENANCY
CRYPTOGRAPHIC_INTEROP: PASS_2_OF_2
WINDOWS_GO_TESTS: PASS_18_PARENTS_68_SUBTESTS
LINUX_GO_TESTS: PASS_36_PARENTS_192_SUBTESTS_1_CHILD_ENTRY_SKIP
GO_RACE_TESTS: PASS_WINDOWS_86_LINUX_228
SPRINT_2R_REGRESSION: PASS_21_CRASH_MATRIX_CASES_PLUS_CR04
HOSTED_CI_BASELINE: PASS_45594545137_45768586579_45768899686
HOSTED_CI_CURRENT_CANDIDATE: UNRUN_UNCOMMITTED
RUNTIME_EVIDENCE_PACKAGE: VERIFIED_30_UNSIGNED_LOGS
OD01_TO_OD08_APPROVED: 0_OF_8
GOVERNANCE_BLOCKERS: 6_OPEN
O0_PRODUCTION_GATE: NO_GO
PRODUCTION_TRUST_INTEGRATION_ALLOWED: NO
PRODUCTION_OPERATOR_INGRESS_ALLOWED: NO
O1_RUNTIME_VERIFIED: NO_FORMAL_GATE_PASS
CLINICAL_PRODUCTION_READY: NO_GO
READY_FOR_SPRINT_3_INDEPENDENT_AUDIT: YES_TEST_ONLY
READY_FOR_SPRINT_4: NO_PENDING_INDEPENDENT_ACCEPTANCE
```
