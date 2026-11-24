// Generated from O0 schemas by pnpm generate.
export type MC01 = {
  contract_id: "MC-01";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  installation_id: string;
  agent_id: string;
  host_binding_sha256: string;
  cert_serial: string;
  key_version: string;
  capabilities: Array<string>;
  status: "ENROLLMENT_PENDING" | "ENROLLED" | "CONNECTED" | "DISCONNECTED" | "ROTATING" | "REVOKED";
  installation_epoch: number;
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC02 = {
  contract_id: "MC-02";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  claim_id: string;
  token_proof: string;
  claim_nonce: string;
  installer_id: string;
  csr_pem: string;
  host_evidence_sha256: string;
  agent_version: string;
  installation_id: string;
  csr_sha256: string;
  key_algorithm: "Ed25519" | "ES256";
};
  signature: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
};
export type MC03 = {
  contract_id: "MC-03";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  job_id: string;
  installation_id: string;
  agent_id: string;
  sequence: number;
  nonce: string;
  idempotency_key: string;
  issued_at: string;
  not_before: string;
  expires_at: string;
  min_agent_version: string;
  approval_grant_ids: Array<string>;
  preconditions: {
  observed_revision?: string;
  maintenance_window_id?: string;
};
} & ({ command_type: "REQUEST_HEALTH"; params: Record<string, never> } | { command_type: "APPLY_SITE_CONFIG"; params: {
  config_sha256: string;
  config_revision: number;
} } | { command_type: "DEPLOY_RELEASE"; params: {
  manifest_sha256: string;
  product_key: string;
  release_channel: "pilot" | "stable" | "hotfix";
} } | { command_type: "VERIFY_DEPLOYMENT"; params: {
  deployment_id: string;
  manifest_sha256: string;
} } | { command_type: "ROTATE_AGENT_CERT"; params: {
  rotation_request_id: string;
  csr_challenge_sha256: string;
} } | { command_type: "APPLY_DEVICE_PROFILE"; params: {
  profile_sha256: string;
  device_id: string;
} } | { command_type: "COLLECT_DIAGNOSTICS"; params: {
  session_id: string;
  diagnostic_scope: "HEALTH" | "GATEWAY_CONNECTIVITY" | "STORAGE";
} } | { command_type: "PAUSE_ROLLOUT"; params: {
  rollout_id: string;
  reason_code: string;
} } | { command_type: "REVOKE_SUPPORT_SESSION"; params: {
  session_id: string;
} });
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC04 = {
  contract_id: "MC-04";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  job_id: string;
  execution_id: string;
  state: "SUCCEEDED" | "FAILED" | "CANCELLED" | "REJECTED" | "UNKNOWN";
  stage: string;
  observed_sha256: string;
  error_code: "NONE" | "APPROVAL_REQUIRED" | "ARTIFACT_UNTRUSTED" | "CONTRACT_INVALID" | "CLOCK_UNTRUSTED" | "EXECUTION_UNKNOWN" | "IDEMPOTENCY_CONFLICT" | "INSTALLATION_SCOPE_MISMATCH" | "JOB_EXPIRED" | "LOCAL_POLICY_DENIED" | "SITE_LOCK_ACTIVE" | "TRUST_REVOKED" | "UNSUPPORTED_COMMAND" | "PRECONDITION_FAILED" | "BACKUP_NOT_VERIFIED" | "OTHER_CODED";
  event_at: string;
  clock_confidence: "TRUSTED" | "UNTRUSTED";
};
  signature: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
};
export type MC05 = {
  contract_id: "MC-05";
  schema_version: string;
  message_id: string;
  sent_at: string;
  body: {
  product_id: string;
  release_id: string;
  topology: "SINGLE_SITE_LAN" | "CENTRAL_CHAIN" | "PRIVATE_CLOUD";
  revision: number;
  components: Array<{
  name: string;
  oci_digest: string;
}>;
  host_os: string;
  minimum_disk_gb: number;
  migration_policy: "NO_DB_CHANGE" | "EXPAND_CONTRACT" | "RESTORE_OR_FORWARD_FIX";
  backup_evidence_required: boolean;
  required_locks: Array<string>;
  health_probes: Array<{
  name: string;
  probe_type: "HTTP" | "TCP" | "SYSTEMD";
  target: string;
}>;
  cpu_arch: "amd64" | "arm64";
  min_memory_mb: number;
  min_agent_version: string;
  expires_at: string;
  sbom_sha256: string;
  migration_set_sha256?: string;
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC06 = {
  contract_id: "MC-06";
  schema_version: string;
  message_id: string;
  sent_at: string;
  body: {
  manufacturer: string;
  exact_model: string;
  software_version: string;
  option_fingerprint_sha256: string;
  adapter_sha256: string;
  capabilities: Array<string>;
  sop_classes: Array<string>;
  measurement_map_version: string;
  fixture_sha256: string;
  certification_state: "DISCOVERED" | "LAB_TESTING" | "PARTIAL" | "VERIFIED" | "MANUAL_ONLY" | "UNSUPPORTED" | "RECALLED";
  transfer_syntaxes: Array<string>;
  dicom_roles: Array<"SCU" | "SCP" | "MWL_SCU" | "MWL_SCP" | "STORAGE_SCU" | "STORAGE_SCP" | "SR_SOURCE">;
  witness_id: string;
  last_verified_at: string;
  adapter_version: string;
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC07 = {
  contract_id: "MC-07";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  agent_id: string;
  component_name: string;
  state: "HEALTHY" | "DEGRADED" | "DOWN" | "UNKNOWN";
  captured_at: string;
  queue_depth: number;
  disk_free_percent: number;
  last_verified_at: string;
  diagnostic_codes: Array<string>;
  dropped_count: number;
  clock_confidence: "TRUSTED" | "UNTRUSTED";
};
  signature: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
};
export type MC08 = {
  contract_id: "MC-08";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  entitlement_id: string;
  product_id: string;
  features: Array<string>;
  issued_at: string;
  expires_at: string;
  offline_grace_days: number;
  policy_epoch: number;
  not_before: string;
  post_grace_mode: "WARN_ONLY" | "READ_ONLY_NON_CLINICAL";
  limits?: {
  site_count?: number;
  seats?: number;
};
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC09 = {
  contract_id: "MC-09";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  grant_id: string;
  operation: "REQUEST_HEALTH" | "APPLY_SITE_CONFIG" | "DEPLOY_RELEASE" | "VERIFY_DEPLOYMENT" | "ROTATE_AGENT_CERT" | "APPLY_DEVICE_PROFILE" | "COLLECT_DIAGNOSTICS" | "PAUSE_ROLLOUT" | "REVOKE_SUPPORT_SESSION" | "ENROL_AGENT" | "AUTHORISE_RECOVERY";
  operator_id: string;
  site_authoriser_id: string;
  resource_scope: Array<string>;
  valid_from: string;
  valid_until: string;
  maker_id: string;
  checker_id: string;
  purpose: "DEPLOYMENT" | "SECURITY_RECOVERY" | "CONFIGURATION_CHANGE" | "REMOTE_SUPPORT";
  target_digest_sha256: string;
  site_authoriser_attestation: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC10 = {
  contract_id: "MC-10";
  schema_version: string;
  message_id: string;
  sent_at: string;
  body: {
  notice_id: string;
  artifact_sha256: string;
  rollout_ring: string;
  incompatibilities: Array<string>;
  clinical_approval_ref: string | null;
  recall_reason: string;
  recovery_plan_version: string;
  notice_revision: number;
  expires_at: string;
  severity: "INFO" | "HIGH" | "CRITICAL";
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC11 = {
  contract_id: "MC-11";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  session_id: string;
  operator_id: string;
  site_authoriser_id: string;
  purpose: string;
  permitted_targets: Array<string>;
  permitted_commands: Array<"READ_TECH_HEALTH" | "VERIFY_GATEWAY_PORTS" | "FETCH_REDACTED_DIAGNOSTIC_CODES" | "RESTART_GATEWAY_APPROVED">;
  valid_from: string;
  valid_until: string;
  recording_policy: string;
  grant_id: string;
  max_duration_s: number;
  max_bytes: number;
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC12 = {
  contract_id: "MC-12";
  schema_version: string;
  message_id: string;
  sent_at: string;
  body: {
  package_type: "FETAL_REFERENCE" | "REPORT_TEMPLATE" | "MEASUREMENT_MAPPING" | "AI_MODEL" | "CLINICAL_RULESET";
  package_sha256: string;
  clinical_owner_signature: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
  intended_use: string;
  reference_version: string;
  validation_evidence_sha256: string;
  valid_from: string;
  valid_until: string;
};
  signature: {
  alg: "Ed25519";
  kid: string;
  value: string;
};
};
export type MC13 = {
  contract_id: "MC-13";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  bundle_id: string;
  manifest_sha256: string;
  scrubber_version: string;
  allowed_metrics: Array<string>;
  retention_days: number;
  phi_exception_grant: string | null;
  session_id: string;
};
  signature: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
};
export type MC14 = {
  contract_id: "MC-14";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  dataset_categories: Array<"APP_CONFIG" | "CLINICAL_DB" | "DICOM_ARCHIVE" | "FINANCIAL_DB" | "MANAGEMENT_AGENT">;
  backup_age_seconds: number;
  last_success_at: string;
  encrypted: boolean;
  integrity_verified: boolean;
  last_restore_test_at: string | null;
  error_state: "NONE" | "BACKUP_FAILED" | "INTEGRITY_FAILED" | "RESTORE_UNTESTED" | "STORAGE_UNAVAILABLE" | "UNKNOWN";
  backup_manifest_sha256: string;
};
  signature: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
};
export type AUX01 = {
  contract_id: "AUX-01";
  schema_version: string;
  message_id: string;
  tenant_id: string;
  site_id: string;
  sent_at: string;
  body: {
  job_id: string;
  progress_state: "RECEIVED" | "VALIDATED" | "ACCEPTED" | "RUNNING";
  stage_code: string;
  event_at: string;
  sequence_number: number;
  execution_id: string;
};
  signature: {
  alg: "Ed25519" | "ES256";
  kid: string;
  value: string;
};
};
export type Envelope = MC01 | MC02 | MC03 | MC04 | MC05 | MC06 | MC07 | MC08 | MC09 | MC10 | MC11 | MC12 | MC13 | MC14 | AUX01;
export interface LocalPayloads {
  locks_response: {
  locked: boolean;
  reasons: Array<string>;
  lease_ttl_s: number;
  revision: string;
};
  readiness_response: {
  ready: boolean;
  service_codes: Array<string>;
  version: string;
};
  profile_stage_request: {
  profile_sha256: string;
  device_id: string;
  site_id: string;
  job_id: string;
};
  configuration_stage_request: {
  config_sha256: string;
  config_revision: number;
  site_id: string;
  job_id: string;
};
  stage_response: {
  status: "STAGED" | "DEFERRED" | "REJECTED";
  reason_code: "NONE" | "SITE_LOCK_ACTIVE" | "INVALID_SIGNATURE" | "UNKNOWN_DEVICE" | "INVALID_CONFIGURATION" | "UNAUTHORISED_PEER" | "OTHER_CODED";
  observed_sha256: string | null;
};
}
