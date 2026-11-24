# OD03 - Private PKI, enrolment and expiry recovery

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Single-use time-bounded claim, CSR proof of possession, HMAC binding to site, installation, nonce and CSR; 90-day cert TTL candidate, renew by day 45; controlled offline re-enrolment requires customer/operator approval and revokes old serial.

**Alternatives considered:**
- Build homemade private CA: REJECT
- Use audited cloud Private CA with KMS: PREFERRED if cloud provider selected
- Audited step-ca + protected keys: ALTERNATIVE if managed CA unavailable

**Open decision / approval dependency:**
Q01 select CA/KMS provider, trust-root rotation, revocation channel and certificate policy

**Accountable approval roles:** Security Authority; Engineering Owner

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
