# OD06 - Product-neutral artifact and deployment lifecycle

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Global artifact signatures independent of customer site; site-scoped MC-03 references immutable manifest digest; generic locks and manifest-defined probes; two different mock products use identical compiled agent. OP17–OP20 runtime delivered O2.

**Alternatives considered:**
- Per-site release artifact re-signing: REJECT
- Immutable globally signed artifacts with per-site scoped job: RECOMMENDED

**Open decision / approval dependency:**
Choose supported OCI/rootless deployment process and product/plugin interface; sign O1/O2 scope

**Accountable approval roles:** Architecture Authority; Product Sponsor

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
