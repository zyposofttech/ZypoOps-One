# OD01 - Authority and clinical independence

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Ops is authoritative only for customers, sites, licences, technical releases and management jobs. ZypoSono is authoritative for clinical information and the local fail-closed clinical activity lock. No routine PHI transfer.

**Alternatives considered:**
- Shared central patient DB: REJECT - violates on-prem data authority and outage independence
- Local clinical database with outbound management-only metadata: RECOMMENDED

**Open decision / approval dependency:**
Q06 local /locks /readiness /profiles /configuration ownership and denial semantics

**Accountable approval roles:** Product Sponsor; Architecture Authority; Clinical Product Owner

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
