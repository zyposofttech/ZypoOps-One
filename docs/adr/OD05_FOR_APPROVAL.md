# OD05 - Operator IAM, tenancy and RLS

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Managed OIDC with phishing-resistant MFA and recovery controls; per-tenant/site permission checks; PostgreSQL composite keys, FORCE RLS and validated per-request transaction-local SET LOCAL. Runtime DB role cannot bypass RLS.

**Alternatives considered:**
- Self-hosted Keycloak from day one: DEFER for operational load
- Managed OIDC with WebAuthn/passkeys: RECOMMENDED

**Open decision / approval dependency:**
Q05 choose actual IdP, break-glass recovery and account lifecycle

**Accountable approval roles:** Security Authority; Architecture Authority

| Named individual | Role / authority | Decision / signature | Date |
|---|---|---|---|
| __________________ | Product/Engineering as applicable | __________ | __________ |
| __________________ | Security/Architecture as applicable | __________ | __________ |
| __________________ | Clinical/Legal as applicable | __________ | __________ |

**Evidence required before formal approval:**
- Relevant source documents and schema hashes recorded
- Threat-model implications and rejected alternatives reviewed
- Owner resolves vendor/retention/procedure values and signs this ADR
- Cross-product interface and rollback/continuity effects accepted
- Real implementation tests remain separately gated; signing an ADR is not a runtime PASS

**Change control:** Once signed, change only through a superseding ADR with migration and compatibility impact. A generated file is never a human signature.
