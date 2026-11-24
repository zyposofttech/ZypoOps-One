PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = FULL;
-- v1.2 proposed agent durability schema. No PHI/clinical data.
CREATE TABLE IF NOT EXISTS local_identity (
 installation_id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, site_id TEXT NOT NULL,
 agent_id TEXT NOT NULL UNIQUE, host_binding_sha256 TEXT NOT NULL,
 cert_issuer TEXT, cert_serial TEXT, cert_not_after TEXT,
 installation_epoch INTEGER NOT NULL DEFAULT 1 CHECK(installation_epoch > 0),
 created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS trusted_key (
 kid TEXT NOT NULL, key_purpose TEXT NOT NULL, algorithm TEXT NOT NULL
 CHECK(algorithm IN ('Ed25519','ES256')),
 public_key_pem TEXT NOT NULL, valid_from TEXT NOT NULL, valid_until TEXT NOT NULL,
 verified_root_revision INTEGER NOT NULL DEFAULT 1 CHECK(verified_root_revision > 0),
 revoked_at TEXT,
 PRIMARY KEY(kid,key_purpose)
);
CREATE TABLE IF NOT EXISTS clock_watermark (
 installation_id TEXT PRIMARY KEY REFERENCES local_identity(installation_id),
 last_trusted_utc TEXT NOT NULL, last_monotonic_ns INTEGER NOT NULL CHECK(last_monotonic_ns >= 0),
 boot_id TEXT NOT NULL, clock_confidence TEXT NOT NULL CHECK(clock_confidence IN ('TRUSTED','UNTRUSTED'))
);
CREATE TABLE IF NOT EXISTS agent_sequence (
 installation_id TEXT PRIMARY KEY REFERENCES local_identity(installation_id),
 last_sequence INTEGER NOT NULL CHECK(last_sequence >= 0),
 sequence_epoch INTEGER NOT NULL DEFAULT 1 CHECK(sequence_epoch > 0),
 updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS received_job (
 job_id TEXT PRIMARY KEY, installation_id TEXT NOT NULL REFERENCES local_identity(installation_id),
 site_id TEXT NOT NULL, command_type TEXT NOT NULL, idempotency_key TEXT NOT NULL,
 payload_sha256 TEXT NOT NULL CHECK(length(payload_sha256)=64), nonce TEXT NOT NULL,
 issuer_kid TEXT NOT NULL, sequence INTEGER NOT NULL CHECK(sequence > 0),
 sequence_epoch INTEGER NOT NULL DEFAULT 1 CHECK(sequence_epoch > 0),
 not_before TEXT NOT NULL, expires_at TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('RECEIVED','VALIDATED','ACCEPTED','RUNNING','SUCCEEDED','FAILED','CANCELLED','REJECTED','UNKNOWN')),
 received_at TEXT NOT NULL, verified_at TEXT,
 UNIQUE(installation_id,command_type,idempotency_key),
 UNIQUE(installation_id,sequence_epoch,sequence),
 UNIQUE(installation_id,nonce)
);
CREATE TABLE IF NOT EXISTS job_stage (
 job_id TEXT NOT NULL REFERENCES received_job(job_id), stage_index INTEGER NOT NULL CHECK(stage_index>=0),
 stage_name TEXT NOT NULL, started_at TEXT NOT NULL, finished_at TEXT,
 state TEXT NOT NULL CHECK(state IN ('STARTED','FINISHED','FAILED','UNKNOWN')),
 helper_receipt_sha256 TEXT, PRIMARY KEY(job_id,stage_index)
);
CREATE TABLE IF NOT EXISTS job_progress_event (
 event_id TEXT PRIMARY KEY, job_id TEXT NOT NULL REFERENCES received_job(job_id),
 progress_state TEXT NOT NULL CHECK(progress_state IN ('RECEIVED','VALIDATED','ACCEPTED','RUNNING')),
 recorded_at TEXT NOT NULL, stage_name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS execution_receipt (
 receipt_id TEXT PRIMARY KEY, job_id TEXT NOT NULL UNIQUE REFERENCES received_job(job_id),
 execution_id TEXT NOT NULL UNIQUE,
 terminal_state TEXT NOT NULL CHECK(terminal_state IN ('SUCCEEDED','FAILED','CANCELLED','REJECTED','UNKNOWN')),
 result_sha256 TEXT NOT NULL CHECK(length(result_sha256)=64), signed_receipt TEXT NOT NULL,
 completed_at TEXT NOT NULL, acknowledged_at TEXT
);
CREATE TABLE IF NOT EXISTS approval_consumption (
 grant_id TEXT PRIMARY KEY, job_id TEXT NOT NULL REFERENCES received_job(job_id),
 grant_sha256 TEXT NOT NULL CHECK(length(grant_sha256)=64), consumed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS configuration_snapshot (
 config_sha256 TEXT PRIMARY KEY CHECK(length(config_sha256)=64),
 active INTEGER NOT NULL DEFAULT 0 CHECK(active IN (0,1)),
 last_good INTEGER NOT NULL DEFAULT 0 CHECK(last_good IN (0,1)), applied_at TEXT
);
CREATE TABLE IF NOT EXISTS offline_entitlement (
 entitlement_id TEXT PRIMARY KEY, signed_payload TEXT NOT NULL,
 policy_epoch INTEGER NOT NULL CHECK(policy_epoch>0), not_before TEXT NOT NULL,
 expires_at TEXT NOT NULL, offline_grace_days INTEGER NOT NULL CHECK(offline_grace_days BETWEEN 0 AND 365),
 post_grace_mode TEXT NOT NULL CHECK(post_grace_mode IN ('WARN_ONLY','READ_ONLY_NON_CLINICAL')),
 last_verified_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS diagnostic_spool (
 event_id TEXT PRIMARY KEY, schema_type TEXT NOT NULL,
 allowlisted_payload TEXT NOT NULL CHECK(length(allowlisted_payload)<=16384),
 expires_at TEXT NOT NULL, retry_count INTEGER NOT NULL DEFAULT 0 CHECK(retry_count>=0)
);
CREATE TABLE IF NOT EXISTS local_action_lock (
 lock_name TEXT PRIMARY KEY, holder_job_id TEXT, acquired_at TEXT NOT NULL, expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_jobs_state ON received_job(state,expires_at);
CREATE INDEX IF NOT EXISTS ix_jobs_site ON received_job(site_id,command_type);
CREATE INDEX IF NOT EXISTS ix_job_progress ON job_progress_event(job_id,recorded_at);
CREATE INDEX IF NOT EXISTS ix_spool_expiry ON diagnostic_spool(expires_at);
