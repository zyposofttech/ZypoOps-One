-- ZOP v1.2 O1 candidate PostgreSQL 16 DDL. Security review + live PG16 tests REQUIRED.
-- No clinical data or routine PHI. Execute only using a dedicated migration role.
-- Application runtime role MUST NOT own the tables or have BYPASSRLS/CREATEROLE.
-- Every tenant-scoped query MUST run in a transaction with validated SET LOCAL app.tenant_id.
-- PostgreSQL RLS is defense in depth, not a substitute for application authorization.
CREATE SCHEMA IF NOT EXISTS zop;
CREATE TABLE IF NOT EXISTS zop.tenant (
 id uuid PRIMARY KEY, legal_name varchar(240) NOT NULL, status varchar(40) NOT NULL
);
CREATE TABLE IF NOT EXISTS zop.site (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES zop.tenant(id), site_code varchar(64) NOT NULL,
 status varchar(40) NOT NULL, UNIQUE(tenant_id,id), UNIQUE(tenant_id,site_code)
);
CREATE TABLE IF NOT EXISTS zop.operator_role (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES zop.tenant(id),
 operator_subject varchar(160) NOT NULL, role_code varchar(60) NOT NULL,
 scope_site_id uuid, valid_until timestamptz, UNIQUE(tenant_id,id),
 FOREIGN KEY(tenant_id,scope_site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.installation (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, installation_epoch integer NOT NULL DEFAULT 1 CHECK(installation_epoch>0),
 status varchar(40) NOT NULL, host_binding_sha256 char(64) NOT NULL, UNIQUE(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.site_agent (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, installation_id uuid NOT NULL,
 agent_version varchar(64) NOT NULL, status varchar(40) NOT NULL, UNIQUE(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id,installation_id) REFERENCES zop.installation(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.enrolment_claim (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, installation_id uuid NOT NULL,
 token_hash char(64) NOT NULL, claim_nonce varchar(128) NOT NULL, issued_at timestamptz NOT NULL,
 expires_at timestamptz NOT NULL, consumed_at timestamptz, consumed_csr_sha256 char(64),
 UNIQUE(tenant_id,site_id,id), UNIQUE(token_hash),
 FOREIGN KEY(tenant_id,site_id,installation_id) REFERENCES zop.installation(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.certificate_identity (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, installation_id uuid NOT NULL,
 issuer varchar(256) NOT NULL, serial varchar(128) NOT NULL, subject_sha256 char(64) NOT NULL,
 not_before timestamptz NOT NULL, not_after timestamptz NOT NULL, revoked_at timestamptz,
 UNIQUE(tenant_id,site_id,id), UNIQUE(issuer,serial),
 FOREIGN KEY(tenant_id,site_id,installation_id) REFERENCES zop.installation(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.revocation_record (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, certificate_id uuid NOT NULL,
 reason_code varchar(60) NOT NULL, revoked_at timestamptz NOT NULL, revocation_epoch bigint NOT NULL CHECK(revocation_epoch>0),
 UNIQUE(tenant_id,site_id,id), UNIQUE(certificate_id),
 FOREIGN KEY(tenant_id,site_id,certificate_id) REFERENCES zop.certificate_identity(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.deployment_artifact (
 id uuid PRIMARY KEY, product_key varchar(80) NOT NULL, artifact_sha256 char(64) NOT NULL UNIQUE,
 signed_manifest_sha256 char(64) NOT NULL, state varchar(40) NOT NULL, created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS zop.management_job (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, agent_id uuid NOT NULL,
 command_type varchar(60) NOT NULL, idempotency_key varchar(160) NOT NULL, nonce varchar(128) NOT NULL,
 job_sequence bigint NOT NULL CHECK(job_sequence>0), payload_sha256 char(64) NOT NULL,
 state varchar(40) NOT NULL, created_at timestamptz NOT NULL, expires_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,command_type,idempotency_key),
 UNIQUE(tenant_id,site_id,nonce),
 FOREIGN KEY(tenant_id,site_id,agent_id) REFERENCES zop.site_agent(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.job_attempt (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, job_id uuid NOT NULL,
 attempt_no integer NOT NULL CHECK(attempt_no>0), state varchar(40) NOT NULL, started_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(job_id,attempt_no),
 FOREIGN KEY(tenant_id,site_id,job_id) REFERENCES zop.management_job(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.job_receipt (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, job_id uuid NOT NULL UNIQUE,
 terminal_state varchar(24) NOT NULL CHECK(terminal_state IN ('SUCCEEDED','FAILED','CANCELLED','REJECTED','UNKNOWN')),
 evidence_sha256 char(64) NOT NULL, completed_at timestamptz NOT NULL, UNIQUE(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id,job_id) REFERENCES zop.management_job(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.job_progress_event (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, job_id uuid NOT NULL,
 stage_code varchar(40) NOT NULL, progress_state varchar(24) NOT NULL,
 event_at timestamptz NOT NULL, sequence_number bigint NOT NULL CHECK(sequence_number>0),
 UNIQUE(tenant_id,site_id,id), UNIQUE(job_id,sequence_number),
 FOREIGN KEY(tenant_id,site_id,job_id) REFERENCES zop.management_job(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.approval_grant (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, target_sha256 char(64) NOT NULL,
 operation varchar(60) NOT NULL, site_authoriser_key_id varchar(120) NOT NULL,
 site_authoriser_attestation_sha256 char(64) NOT NULL, ops_grant_sha256 char(64) NOT NULL,
 maker_id varchar(160) NOT NULL, checker_id varchar(160) NOT NULL,
 valid_from timestamptz NOT NULL, valid_until timestamptz NOT NULL,
 consumed_at timestamptz, UNIQUE(tenant_id,site_id,id), CHECK(maker_id<>checker_id),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.job_approval_grant (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, job_id uuid NOT NULL, grant_id uuid NOT NULL,
 UNIQUE(job_id,grant_id), UNIQUE(grant_id),
 FOREIGN KEY(tenant_id,site_id,job_id) REFERENCES zop.management_job(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id,grant_id) REFERENCES zop.approval_grant(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.site_authoriser_key (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 key_id varchar(120) NOT NULL, algorithm varchar(16) NOT NULL CHECK(algorithm IN ('Ed25519','ES256')),
 public_key_sha256 char(64) NOT NULL, valid_until timestamptz NOT NULL, revoked_at timestamptz,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,key_id),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.action_lock (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, lock_name varchar(80) NOT NULL,
 holder_job_id uuid, acquired_at timestamptz NOT NULL, expires_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,lock_name),
 FOREIGN KEY(tenant_id,site_id,holder_job_id) REFERENCES zop.management_job(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.site_configuration_revision (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 config_sha256 char(64) NOT NULL, revision_no bigint NOT NULL CHECK(revision_no>0),
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,revision_no),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.entitlement (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 product_key varchar(80) NOT NULL, policy_epoch bigint NOT NULL CHECK(policy_epoch>0),
 entitlement_sha256 char(64) NOT NULL, expires_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,product_key,policy_epoch),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.rollout_attempt (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, artifact_sha256 char(64) NOT NULL,
 state varchar(40) NOT NULL, UNIQUE(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.health_sample (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 component_code varchar(50) NOT NULL, numeric_value numeric, captured_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.operational_audit_event (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 event_seq bigint NOT NULL CHECK(event_seq>0), event_code varchar(60) NOT NULL,
 actor_subject varchar(160) NOT NULL, event_sha256 char(64) NOT NULL,
 previous_chain_sha256 char(64) NOT NULL, created_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,event_seq),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.audit_anchor (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 chain_head_sha256 char(64) NOT NULL, through_event_seq bigint NOT NULL,
 external_immutable_object_uri varchar(1024) NOT NULL, anchored_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,through_event_seq),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.support_session (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, approval_grant_id uuid NOT NULL,
 status varchar(32) NOT NULL, expires_at timestamptz NOT NULL, UNIQUE(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id,approval_grant_id) REFERENCES zop.approval_grant(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.support_diagnostic_bundle (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, session_id uuid NOT NULL,
 content_sha256 char(64) NOT NULL, state varchar(32) NOT NULL,
 UNIQUE(tenant_id,site_id,id), FOREIGN KEY(tenant_id,site_id,session_id) REFERENCES zop.support_session(tenant_id,site_id,id)
);
CREATE TABLE IF NOT EXISTS zop.backup_status (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 backup_manifest_sha256 char(64) NOT NULL, status varchar(32) NOT NULL, last_verified_at timestamptz,
 UNIQUE(tenant_id,site_id,id), FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.restore_drill (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, evidence_sha256 char(64) NOT NULL,
 state varchar(32) NOT NULL, UNIQUE(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.outbox_event (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 event_type varchar(60) NOT NULL, body_sha256 char(64) NOT NULL,
 created_at timestamptz NOT NULL, published_at timestamptz,
 UNIQUE(tenant_id,site_id,id), FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.site_ownership_event (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 ownership_revision bigint NOT NULL, event_code varchar(60) NOT NULL, recorded_at timestamptz NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,ownership_revision),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
CREATE TABLE IF NOT EXISTS zop.signing_attestation (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL,
 purpose varchar(80) NOT NULL, key_id varchar(120) NOT NULL, target_sha256 char(64) NOT NULL,
 created_at timestamptz NOT NULL, UNIQUE(tenant_id,site_id,id),
 FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id)
);
-- Tenant RLS; SELECT/INSERT/UPDATE/DELETE policy evaluated against a *validated* transaction-local tenant.
ALTER TABLE zop.tenant ENABLE ROW LEVEL SECURITY;
ALTER TABLE zop.tenant FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_identity ON zop.tenant
 USING(id=nullif(current_setting('app.tenant_id',true),'')::uuid)
 WITH CHECK(id=nullif(current_setting('app.tenant_id',true),'')::uuid);
-- Apply RLS automatically to every table carrying tenant_id except global deployment_artifact.
DO $plpgsql$ DECLARE r record;
BEGIN
 FOR r IN SELECT table_name FROM information_schema.columns
     WHERE table_schema='zop' AND column_name='tenant_id' GROUP BY table_name
 LOOP
  EXECUTE format('ALTER TABLE zop.%I ENABLE ROW LEVEL SECURITY',r.table_name);
  EXECUTE format('ALTER TABLE zop.%I FORCE ROW LEVEL SECURITY',r.table_name);
  EXECUTE format('CREATE POLICY tenant_isolation ON zop.%I USING (tenant_id = nullif(current_setting(''app.tenant_id'',true),'''')::uuid) WITH CHECK (tenant_id = nullif(current_setting(''app.tenant_id'',true),'''')::uuid)',r.table_name);
 END LOOP;
END $plpgsql$;
-- Production roles: dedicated non-owner runtime role with NOBYPASSRLS and only required DML;
-- separate tightly controlled migrations/operational support roles. No owner/DDL privilege in API.
-- To guard the append-only audit table, REVOKE UPDATE, DELETE on it from the API role.
-- Hourly signed chain-head export to a separate WORM account remains runtime work.
-- PostgreSQL + Prisma integration / pooled concurrency + raw SQL bypass tests NOT EXECUTED.
