# O1 Sprint 3 internal service catalogue

This reference describes the synthetic NestJS application returned by `createSyntheticApplication(config)`. Each app instance owns its principal objects and PostgreSQL pools. Call `app.close()` when replacing it. After reopening, authenticate again. Production ingress requires OD05/Q05.

## Caller and scope

`SyntheticContext.authenticate(token)` accepts a separately provisioned 32-byte, hex-encoded credential and returns the corresponding frozen `Identity`. Its tenant, site, actor and role come from trusted configuration. An Agent identity also pins one installation and one Agent. Forged objects, credentials from another application instance and missing identities fail closed. `NODE_ENV=test`, loopback database URLs, the expected non-owner roles and explicit configuration are required. The test transport uses an inherited private pipe.

## Registry and enrolment

| Operation | Caller | Inputs and result |
|---|---|---|
| `RegistryService.createSite` | Provisioner | Site code, timezone, deployment profile, data location, connectivity and maintenance references. Optional legal name and customer details create the scoped customer and legal entity in the same transaction. Returns the site ID. |
| `updateCustomer` | Provisioner | Changes contract, tier, contact, approver and deployment-constraint references and lifecycle. Returns the stored customer. EXITED preserves installations and evidence. |
| `register` | Provisioner | Installation UUID, host digest, product and optional predecessor UUID. Returns the new scoped installation and epoch. Each registration requires a fresh identity. |
| `transferOwnership` | Provisioner | Current site UUID. Returns SIGNED_TRANSFER_CEREMONY_REQUIRED. |
| `enrolAuthoriser` | Provisioner | Site authoriser UUID and trusted public key ID. Pins signer identity, scope, algorithm and fingerprint. |
| `inventory` | Operator or provisioner | Optional exclusive installation UUID cursor and limit 1–100. Returns scoped customer, legal entities, site and installations. |
| `EnrolmentService.claim` | Operator | Installation, intended installer and TTL 1–600 seconds. Returns claim ID, nonce, expiry and a bootstrap secret for delivery to that installer. |
| `consume` | Installer | Signed MC-02 wire. Checks intended installer, CSR signature, enrolled PoP public key, CSR digest, HMAC, nonce, host, installation, scope, time and atomic consumption. Returns signed synthetic MC-01. |
| `revoke` | Operator | Agent UUID. Records the revocation, retires the identity and invalidates its site signing keys in one transaction. |

Customer details use `{contract,tier,contact,approver,constraint}`. An Agent key ID alone cannot claim an installation. The synthetic CSR verifier supports Ed25519. Production certificate issuance and re-enrolment ceremonies remain under OD02/OD03.

## Catalogue and approvals

| Operation | Caller | Inputs and result |
|---|---|---|
| `CatalogueService.publish` | Provisioner | Global signed MC-05, optional edition and dependency references. Validates the manifest and fixed simulator readiness endpoint. Returns its canonical SHA-256. An identical repeat returns the same digest. A conflicting publication fails. |
| `artifact` | Operator or provisioner | Exact digest. Returns the stored signed manifest, edition, dependency metadata and channel. |
| `entitlement` | Operator | Scoped signed MC-08. Persists a strictly increasing product revision and its exact signed policy. The synthetic path supports WARN_ONLY. |
| `request` | Operator acting as maker | Target digest and DEPLOY_RELEASE or APPLY_SITE_CONFIG. Returns an approval request ID. |
| `approve` | A different operator acting as checker | Signed MC-09 including the independently signed site authorisation. Verifies both signatures, authoriser registry, maker, checker, validity, operation, resource and target, then stores the grant. |
| `revokeGrant` | Operator | Grant ID. Records revocation and a scoped audit event. |

The site authority retains its private key. The service receives the completed dual-proof MC-09. Catalogue dependencies and edition are immutable operator metadata alongside the signed MC-05. The pilot channel is the connected simulator's supported channel.

## Readiness, signed evidence and dispatch

| Operation | Caller | Inputs and result |
|---|---|---|
| `ReadinessService.plan` | Operator | Installation, artifact, expected backup manifest digest, dataset categories and typed assessment. Returns plan ID, readiness, hard blockers and connectivity warnings. |
| `stage` | Operator | Expected prior revision, `{pollSeconds,profile,secretRef}` and approved configuration grant. Returns revision, canonical digest, changed field names and the local application gate. |
| `EvidenceService.backup` | Agent | Plan ID and signed MC-14. Checks signer, installation, operation-specific manifest, exact categories, age, timestamps, encryption and integrity. Stores technical evidence. |
| `health` | Agent | Signed MC-07. Checks identity, enrolled key, clock, allowed component/diagnostic codes and per-component ingest interval. Stores the permitted technical sample. |
| `status` | Operator | Agent and product. Returns identity, version, last known sample, current health projection and entitlement state. Missing or stale verification yields UNKNOWN or STALE. |
| `JobService.create` | Operator | Plan ID, one to eight distinct grant UUIDs and UUID idempotency key. Returns the same job ID for an identical retry. Changed inputs conflict. The signed bundle, grant associations, installation sequence and outbox row commit atomically. |
| `next` | Agent | Uses the authenticated installation and Agent. Returns at most one immutable bundle string or null. Acquires the site lock, tracks attempts and checks current dispatch authority. |
| `progress` | Agent | Signed AUX-01 with original execution ID and next sequence. An identical last event can be redelivered. A terminal job rejects subsequent progress. |
| `receipt` | Agent | Signed MC-04. Joins original command and independently enrolled receipt key to check identity, execution, artifact, stage, outcome and error code. Stores original terminal bytes and projects state atomically. Exact canonical redelivery is idempotent. |
| `deliveryStatus` | Operator | Job ID. Returns persisted state, attempt count, delivery projection and signed terminal wire when available. An unresolved delivered expiry requires reconciliation. |
| `OnboardingService.assess` | Operator | Plan ID. Returns customer/site/installation, Agent, entitlement, prerequisite, artifact, backup, grant and verified-outcome checks. Technical verification leaves production and clinical status at NO_GO. |
| `history` | Operator | Exclusive event sequence and page size 1–100. Returns chain-checked scoped entries, cursor, head digest and external anchor status. |

The plan assessment records CPU architecture, RAM, disk headroom, OS, trusted-clock status, connectivity, supported profile and backup readiness. Signed backup evidence is checked again at job admission and by the local roles.

## Errors and retry decisions

| Error or state | Caller action |
|---|---|
| UNAUTHENTICATED, FORBIDDEN, SIGNER_BINDING, CSR_BINDING, TOKEN_PROOF | Correct the independently provisioned identity or proof. Authentication or signing secrets do not belong in logs. |
| CLAIM_UNAVAILABLE, IDENTITY_UNAVAILABLE | Check claim expiry, prior consumption or identity revocation. |
| APPROVAL_DENIED, CONSENT_BINDING, GRANT_BINDING, AUTHORITY_EXPIRED, BACKUP_UNAVAILABLE | Obtain the required current proof for this scope and target. |
| IDEMPOTENCY_CONFLICT, ARTIFACT_CONFLICT, REVISION_CONFLICT, PROGRESS_SEQUENCE, OUTCOME_CONFLICT | Inspect the existing durable operation before retrying. |
| SITE_HOLD, RECONCILIATION_REQUIRED | Reconcile the original operation. Sign-off for production hold release remains outstanding. |
| OPERATION_REJECTED | A bounded response for a database or other internal rejection. Once the site exists, the authenticated rejected operation can be correlated through its scoped audit history. |

The central transport sends a maximum of one bundle per poll. Redelivery has a 250 ms minimum interval and a 20-attempt cap. The Agent's signed terminal result controls the outcome. A future console needs an approved operator identity, ingress, authorisation and redaction design before exposing these operations.
