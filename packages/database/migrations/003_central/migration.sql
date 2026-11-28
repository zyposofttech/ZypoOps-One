BEGIN;
ALTER TABLE zop.tenant
  ADD COLUMN contract_ref varchar(160),
  ADD COLUMN support_tier varchar(160),
  ADD COLUMN contact_ref varchar(160),
  ADD COLUMN approver_ref varchar(160),
  ADD COLUMN deployment_constraint varchar(160);
ALTER TABLE zop.site
  ADD COLUMN timezone varchar(160),
  ADD COLUMN deployment_profile varchar(160),
  ADD COLUMN data_location varchar(160),
  ADD COLUMN connectivity varchar(160),
  ADD COLUMN maintenance varchar(160);
ALTER TABLE zop.installation
  ADD COLUMN product_key varchar(160),
  ADD COLUMN replaces_id uuid;
ALTER TABLE zop.site_agent
  ADD COLUMN receipt_kid varchar(160),
  ADD COLUMN identity_wire text CHECK (octet_length(identity_wire) <= 1048576);
ALTER TABLE zop.site_signing_key
  ADD COLUMN agent_id uuid;
ALTER TABLE zop.enrolment_claim
  ADD COLUMN installer_id uuid;
ALTER TABLE zop.deployment_artifact
  ADD COLUMN edition varchar(160),
  ADD COLUMN service_dependencies text CHECK (octet_length(service_dependencies)<=4096),
  ADD COLUMN signed_wire text CHECK (octet_length(signed_wire) <= 1048576),
  ADD COLUMN release_id varchar(160),
  ADD COLUMN channel varchar(160);
ALTER TABLE zop.entitlement
  ADD COLUMN signed_wire text CHECK (octet_length(signed_wire) <= 1048576);
ALTER TABLE zop.approval_grant
  ADD COLUMN signed_wire text CHECK (octet_length(signed_wire) <= 1048576),
  ADD COLUMN revoked_at timestamptz;
ALTER TABLE zop.management_job
  ADD COLUMN bundle_wire text CHECK (octet_length(bundle_wire) <= 1048576),
  ADD COLUMN execution_id uuid,
  ADD COLUMN installation_id uuid,
  ADD COLUMN request_sha256 char(64);
ALTER TABLE zop.job_receipt
  ADD COLUMN signed_wire text CHECK (octet_length(signed_wire) <= 1048576);
ALTER TABLE zop.job_progress_event
  ADD COLUMN signed_wire text CHECK (octet_length(signed_wire) <= 1048576);
ALTER TABLE zop.health_sample
  ADD COLUMN agent_id uuid,
  ADD COLUMN signed_wire text CHECK (octet_length(signed_wire) <= 1048576);
ALTER TABLE zop.backup_status
  ADD COLUMN agent_id uuid,
  ADD COLUMN operation_id uuid,
  ADD COLUMN target_sha256 char(64),
  ADD COLUMN signed_wire text CHECK (octet_length(signed_wire) <= 1048576);
ALTER TABLE zop.site_configuration_revision
  ADD COLUMN settings_wire text CHECK (octet_length(settings_wire) <= 1048576),
  ADD COLUMN approval_id uuid;
ALTER TABLE zop.operational_audit_event
  ADD COLUMN resource_id uuid,
  ADD COLUMN correlation_id uuid,
  ADD COLUMN outcome varchar(160),
  ADD COLUMN reason_code varchar(160);
ALTER TABLE zop.management_job ADD CONSTRAINT installation_sequence UNIQUE(tenant_id,site_id,installation_id,job_sequence),
 ADD FOREIGN KEY(tenant_id,site_id,installation_id) REFERENCES zop.installation(tenant_id,site_id,id);
ALTER TABLE zop.site_agent ADD UNIQUE(tenant_id,site_id,receipt_kid);
CREATE UNIQUE INDEX single_active_agent ON zop.site_agent(installation_id) WHERE status='ENROLLED';
ALTER TABLE zop.deployment_artifact ADD UNIQUE(product_key,release_id,channel);
ALTER TABLE zop.installation ADD FOREIGN KEY(tenant_id,site_id,replaces_id) REFERENCES zop.installation(tenant_id,site_id,id);
ALTER TABLE zop.site_signing_key ADD FOREIGN KEY(tenant_id,site_id,agent_id) REFERENCES zop.site_agent(tenant_id,site_id,id);
ALTER TABLE zop.health_sample ADD FOREIGN KEY(tenant_id,site_id,agent_id) REFERENCES zop.site_agent(tenant_id,site_id,id);
ALTER TABLE zop.backup_status ADD FOREIGN KEY(tenant_id,site_id,agent_id) REFERENCES zop.site_agent(tenant_id,site_id,id);
ALTER TABLE zop.site_configuration_revision ADD FOREIGN KEY(tenant_id,site_id,approval_id) REFERENCES zop.approval_grant(tenant_id,site_id,id);
GRANT INSERT ON zop.enrolment_claim,zop.site_agent,zop.certificate_identity,zop.site_signing_key,zop.revocation_record,zop.entitlement,zop.backup_status,zop.signing_attestation TO zop_runtime;
GRANT UPDATE(consumed_at,consumed_csr_sha256) ON zop.enrolment_claim TO zop_runtime;
GRANT UPDATE(status) ON zop.site_agent TO zop_runtime;
GRANT UPDATE(revoked_at) ON zop.certificate_identity,zop.site_signing_key,zop.approval_grant TO zop_runtime;
-- Only the test harness creates this optional, non-owner login. All scoped tables retain FORCE RLS.
DO $$ BEGIN
 IF EXISTS (SELECT FROM pg_roles WHERE rolname='zop_synthetic_provision') THEN
  GRANT USAGE ON SCHEMA zop TO zop_synthetic_provision;
  GRANT SELECT,INSERT ON zop.tenant,zop.customer_legal_entity,zop.site,zop.installation,zop.deployment_artifact,zop.operational_audit_event,zop.site_authoriser_key TO zop_synthetic_provision;
  GRANT UPDATE(status,contract_ref,support_tier,contact_ref,approver_ref,deployment_constraint) ON zop.tenant TO zop_synthetic_provision;
 END IF;
END $$;
-- Stage and consume together. Rolled-back revisions leave the grant available.
CREATE FUNCTION zop.consume_config_grant() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
BEGIN
 IF NEW.approval_id IS NULL THEN RETURN NEW; END IF;
 UPDATE zop.approval_grant SET consumed_at=statement_timestamp()
 WHERE id=NEW.approval_id AND tenant_id=NEW.tenant_id AND site_id=NEW.site_id AND operation='APPLY_SITE_CONFIG'
 AND target_sha256=NEW.config_sha256 AND consumed_at IS NULL AND revoked_at IS NULL
 AND valid_from<=statement_timestamp() AND valid_until>statement_timestamp();
 IF NOT FOUND THEN RAISE EXCEPTION 'CONFIG_APPROVAL_UNAVAILABLE'; END IF;
 RETURN NEW;
END $$;
REVOKE ALL ON FUNCTION zop.consume_config_grant() FROM PUBLIC;
CREATE TRIGGER consumed_config BEFORE INSERT ON zop.site_configuration_revision FOR EACH ROW EXECUTE FUNCTION zop.consume_config_grant();
CREATE UNIQUE INDEX single_staged_config_per_grant ON zop.site_configuration_revision(approval_id) WHERE approval_id IS NOT NULL;
CREATE TABLE zop.deployment_plan (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL, site_id uuid NOT NULL, installation_id uuid NOT NULL,
 artifact_sha256 char(64) NOT NULL REFERENCES zop.deployment_artifact(artifact_sha256),
 backup_digest char(64) NOT NULL, categories varchar(40)[] NOT NULL,
 assessment text NOT NULL CHECK(octet_length(assessment)<=4096), ready boolean NOT NULL,
 UNIQUE(tenant_id,site_id,id), UNIQUE(tenant_id,site_id,backup_digest),
 FOREIGN KEY(tenant_id,site_id,installation_id) REFERENCES zop.installation(tenant_id,site_id,id));
ALTER TABLE zop.deployment_plan ENABLE ROW LEVEL SECURITY;
ALTER TABLE zop.deployment_plan FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON zop.deployment_plan USING
 (tenant_id=nullif(current_setting('app.tenant_id',true),'')::uuid AND site_id=nullif(current_setting('app.site_id',true),'')::uuid)
 WITH CHECK (tenant_id=nullif(current_setting('app.tenant_id',true),'')::uuid AND site_id=nullif(current_setting('app.site_id',true),'')::uuid);
ALTER TABLE zop.backup_status ADD FOREIGN KEY(tenant_id,site_id,operation_id) REFERENCES zop.deployment_plan(tenant_id,site_id,id);
GRANT SELECT,INSERT ON zop.deployment_plan TO zop_runtime;
COMMIT;
