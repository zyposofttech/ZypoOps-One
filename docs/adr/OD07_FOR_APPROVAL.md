# OD07 - Management telemetry and privacy separation

**STATUS: PROPOSED - HUMAN APPROVAL REQUIRED**
**Candidate decision (NOT approved):**
Fixed-code technical metadata only, bounded schemas, no routine patient info. Exceptional PHI transfer is separately authorised, encrypted, minimum-necessary, time-limited, access-logged and prohibited by default pending policy.

**Alternatives considered:**
- Forward generic logs/raw DICOM diagnostics to cloud: REJECT
- Strict allow-list fixed-code telemetry and controlled local diagnostics: RECOMMENDED

**Open decision / approval dependency:**
Q07 decide retention by data class and any lawful exception basis with privacy/legal review

**Accountable approval roles:** Privacy/Legal Owner; Clinical Owner

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
