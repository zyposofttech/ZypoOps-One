# OD02 - Agent runtime and protected identity

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Ubuntu 24.04 LTS, systemd, Go executable, SQLite WAL; per-installation identity, TPM2 ES256 when available. Non-TPM software key fallback may be approved only under documented risk controls and restricted high-risk operations.

**Alternatives considered:**
- Node/Python all-in-one agent: REJECT - runtime and privilege footprint
- Go agent with TPM2 ES256 preferred: RECOMMENDED
- Ed25519 software key with secure OS permissions: CONDITIONAL fallback under local approval

**Open decision / approval dependency:**
Q02 fallback allowed host classes, key custody and replacement ceremony

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
