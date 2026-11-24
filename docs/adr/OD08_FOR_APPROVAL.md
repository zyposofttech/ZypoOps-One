# OD08 - Offline entitlement and continuity

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Local clinical work survives cloud/internet outage and licence expiration if local site infrastructure operates. Candidate 90-day offline grace and WARN_ONLY after grace; no destructive deletion, report lock or forced denial of lawful historical records.

**Alternatives considered:**
- Online heartbeat required for clinical access: REJECT
- Signed offline entitlement with non-destructive post-grace mode: RECOMMENDED

**Open decision / approval dependency:**
Q03 approve actual grace duration, commercial policy, clock recovery and regulatory exception process

**Accountable approval roles:** Product Sponsor; Privacy/Legal Owner; Clinical Owner

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
