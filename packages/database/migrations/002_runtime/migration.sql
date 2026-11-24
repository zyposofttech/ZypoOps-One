BEGIN;
CREATE TABLE zop.customer_legal_entity (
  id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES zop.tenant(id),
  legal_name varchar(240) NOT NULL, UNIQUE (tenant_id,id)
);
CREATE TABLE zop.trusted_signing_key (
  kid varchar(80) PRIMARY KEY, purpose varchar(80) NOT NULL,
  algorithm varchar(16) NOT NULL CHECK (algorithm IN ('Ed25519','ES256')),
  public_key_pem text NOT NULL CHECK (length(public_key_pem) <= 4096),
  root_revision bigint NOT NULL CHECK (root_revision > 0),
  valid_from timestamptz NOT NULL, valid_until timestamptz NOT NULL,
  revoked_at timestamptz, CHECK (valid_until > valid_from)
);
-- Site-originated identities have a separate namespace and a mandatory ownership FK.
CREATE TABLE zop.site_signing_key (
  tenant_id uuid NOT NULL, site_id uuid NOT NULL, kid varchar(80) NOT NULL,
  purpose varchar(80) NOT NULL, algorithm varchar(16) NOT NULL CHECK (algorithm IN ('Ed25519','ES256')),
  public_key_pem text NOT NULL CHECK (length(public_key_pem) <= 4096),
  root_revision bigint NOT NULL CHECK (root_revision > 0),
  valid_from timestamptz NOT NULL, valid_until timestamptz NOT NULL, revoked_at timestamptz,
  PRIMARY KEY(tenant_id,site_id,kid), FOREIGN KEY(tenant_id,site_id) REFERENCES zop.site(tenant_id,id),
  CHECK (valid_until > valid_from)
);
ALTER TABLE zop.rollout_attempt ADD FOREIGN KEY(artifact_sha256) REFERENCES zop.deployment_artifact(artifact_sha256);
ALTER TABLE zop.management_job ADD UNIQUE(tenant_id,site_id,agent_id,job_sequence),
  ADD COLUMN target_sha256 char(64) CHECK(target_sha256 ~ '^[0-9a-f]{64}$');
ALTER TABLE zop.approval_grant ADD CHECK(valid_until > valid_from);
ALTER TABLE zop.job_approval_grant ADD COLUMN consumed_at timestamptz NOT NULL DEFAULT transaction_timestamp();
DO $$
DECLARE r record; predicate text;
BEGIN
 FOR r IN SELECT table_name FROM information_schema.columns WHERE table_schema='zop' AND column_name='tenant_id' LOOP
  predicate := 'tenant_id = nullif(current_setting(''app.tenant_id'',true),'''')::uuid';
  IF r.table_name='site' THEN predicate := predicate || ' AND id = nullif(current_setting(''app.site_id'',true),'''')::uuid';
  ELSIF r.table_name='operator_role' THEN predicate := predicate || ' AND (scope_site_id IS NULL OR scope_site_id = nullif(current_setting(''app.site_id'',true),'''')::uuid)';
  ELSIF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='zop' AND table_name=r.table_name AND column_name='site_id') THEN
   predicate := predicate || ' AND site_id = nullif(current_setting(''app.site_id'',true),'''')::uuid';
  END IF;
  EXECUTE format('ALTER TABLE zop.%I ENABLE ROW LEVEL SECURITY',r.table_name);
  EXECUTE format('ALTER TABLE zop.%I FORCE ROW LEVEL SECURITY',r.table_name);
  EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON zop.%I',r.table_name);
  EXECUTE format('CREATE POLICY tenant_isolation ON zop.%I USING (%s) WITH CHECK (%s)',r.table_name,predicate,predicate);
 END LOOP;
END $$;
CREATE FUNCTION zop.guard_job_transition() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='INSERT' THEN
   IF NEW.state <> 'RECEIVED' THEN RAISE EXCEPTION 'JOB_INITIAL_STATE'; END IF;
 ELSIF NEW.state IS DISTINCT FROM OLD.state AND NOT (
   (OLD.state='RECEIVED' AND NEW.state IN ('VALIDATED','REJECTED','CANCELLED')) OR
   (OLD.state='VALIDATED' AND NEW.state IN ('ACCEPTED','REJECTED','CANCELLED')) OR
   (OLD.state='ACCEPTED' AND NEW.state IN ('RUNNING','CANCELLED','REJECTED')) OR
   (OLD.state='RUNNING' AND NEW.state IN ('SUCCEEDED','FAILED','UNKNOWN','CANCELLED'))
 ) THEN RAISE EXCEPTION 'JOB_TRANSITION'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER job_state BEFORE INSERT OR UPDATE ON zop.management_job FOR EACH ROW EXECUTE FUNCTION zop.guard_job_transition();

CREATE FUNCTION zop.consume_grant() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
BEGIN
 UPDATE zop.approval_grant g SET consumed_at=statement_timestamp()
 FROM zop.management_job j WHERE g.id=NEW.grant_id AND j.id=NEW.job_id
 AND (g.tenant_id,g.site_id)=(NEW.tenant_id,NEW.site_id) AND (j.tenant_id,j.site_id)=(g.tenant_id,g.site_id)
 AND g.operation=j.command_type AND g.target_sha256=j.target_sha256 AND g.consumed_at IS NULL
 AND g.valid_from <= statement_timestamp() AND g.valid_until > statement_timestamp()
 AND j.expires_at > statement_timestamp() AND j.state IN ('RECEIVED','VALIDATED','ACCEPTED');
 IF NOT FOUND THEN RAISE EXCEPTION 'GRANT_UNAVAILABLE'; END IF;
 NEW.consumed_at=statement_timestamp();
 RETURN NEW;
END $$;
CREATE TRIGGER grant_single_use BEFORE INSERT ON zop.job_approval_grant FOR EACH ROW EXECUTE FUNCTION zop.consume_grant();
REVOKE ALL ON FUNCTION zop.consume_grant() FROM PUBLIC;
GRANT USAGE ON SCHEMA zop TO zop_runtime;
GRANT SELECT ON ALL TABLES IN SCHEMA zop TO zop_runtime;
REVOKE ALL ON zop._prisma_migrations FROM zop_runtime;
GRANT INSERT ON zop.management_job,zop.job_attempt,zop.job_receipt,zop.job_progress_event,
 zop.approval_grant,zop.job_approval_grant,zop.health_sample,zop.operational_audit_event,
 zop.action_lock,zop.site_configuration_revision,zop.outbox_event,zop.rollout_attempt TO zop_runtime;
GRANT UPDATE(state) ON zop.management_job,zop.job_attempt,zop.rollout_attempt TO zop_runtime;
ALTER TABLE zop.job_approval_grant ADD UNIQUE(tenant_id,site_id,grant_id);
ALTER TABLE zop.job_receipt ADD UNIQUE(tenant_id,site_id,job_id);
ALTER TABLE zop.revocation_record ADD UNIQUE(tenant_id,site_id,certificate_id);
COMMIT;
