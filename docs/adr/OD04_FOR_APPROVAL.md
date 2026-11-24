# OD04 - Management job signing and dual customer consent

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Use wrapped MC-03, RFC 8785 JCS signing profile with per-contract domain separation and key-purpose pinning. MC-09 requires site-authoriser signature binding tenant+site+body+target. Agent and helper independently verify. Rootless Podman/Quadlet templates; no arbitrary commands.

**Alternatives considered:**
- Ops-only approval for high-risk tasks: REJECT
- Dual Ops + site authoriser cryptographic attestation: RECOMMENDED

**Open decision / approval dependency:**
Q04 choose supported TUF implementation, supported package metadata and key custody

**Accountable approval roles:** Security Authority; Product Sponsor

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
