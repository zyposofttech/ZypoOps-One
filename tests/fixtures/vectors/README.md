v1.2 TEST-ONLY vectors. PRIVATE KEYS ARE GENERATED EPHEMERALLY AND NEVER EXPORTED. All public keys are marked test-only.
Site approval signature = ASCII("ZOP-SITE-CONSENT-V1\x00") || JCS({tenant_id, site_id, body: body_excluding_attestation}).
Envelope signature = ASCII("ZOP-SIG-V1\x00<contract_id>\x00") || JCS(envelope_excluding_signature).
MC-12 clinical owner signs "ZOP-CLINICAL-PACKAGE-V1\x00" + JCS(MC-12.body excluding clinical_owner_signature).
All production verifiers MUST require a trust-anchor validated key registry, per-key purpose, site scope, non-revoked validity and canonical parsing.
Node/Go included scripts test restricted ASCII-compatible fixtures, NOT full RFC 8785 Unicode/IEEE754 conformance.
No test key in this pack is authorised in any real deployment; no live CA/TPM/KMS access or signatures are provided.
