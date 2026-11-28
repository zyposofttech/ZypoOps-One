import {createPublicKey} from 'node:crypto';
import {verifyMessage} from '../../../packages/signing-ts/index.ts';
import {SyntheticContext, activeAgent, audit, canon, requireThat, scope, uuid, type Identity, type Tx} from './context.ts';

export async function boundKey(ctx:SyntheticContext,tx:Tx,p:Identity,kid:string,purpose:string) {
  const now=new Date(), row=await tx.site_signing_key.findFirst({where:{...scope(p),kid,purpose,agent_id:p.agent,valid_from:{lte:now},valid_until:{gt:now},revoked_at:null}});
  const key=ctx.config.trust.get(kid);
  requireThat(row && key && !key.revoked && key.publicKey.equals(createPublicKey(row.public_key_pem)),'SIGNER_BINDING');
}
export class EvidenceService {
  readonly ctx:SyntheticContext;
  constructor(ctx:SyntheticContext) {this.ctx=ctx;}
  backup(p:Identity,id:string,wire:string) {
    this.ctx.authorise(p,['agent']);uuid(id);
    const m=verifyMessage(wire,{contract:'MC-14',scope:scope(p)},this.ctx.config.trust);
    requireThat(m.contract_id==='MC-14','BACKUP_CONTRACT');
    const b=m.body,sent=Date.parse(m.sent_at),success=Date.parse(b.last_success_at),age=Date.now()-success;
    requireThat(sent<=Date.now() && success<=sent && age>=0 && age<86400000 && Math.floor((sent-success)/1000)===b.backup_age_seconds && b.encrypted&&b.integrity_verified&&b.error_state==='NONE','BACKUP_UNAVAILABLE');
    return this.ctx.database.run(p,async tx=>{
      await activeAgent(tx,p);await boundKey(this.ctx,tx,p,m.signature.kid,'agent_backup_receipt');
      const plan=await tx.deployment_plan.findFirstOrThrow({where:{id,installation_id:p.installation}});
      requireThat(plan.backup_digest===b.backup_manifest_sha256 && canon([...plan.categories].sort())===canon([...b.dataset_categories].sort()),'BACKUP_BINDING');
      await tx.backup_status.create({data:{...scope(p),id:m.message_id,agent_id:p.agent,operation_id:plan.id,target_sha256:plan.artifact_sha256,signed_wire:canon(m),backup_manifest_sha256:b.backup_manifest_sha256,status:'VERIFIED',last_verified_at:new Date(success)}});
      await audit(tx,p,'BACKUP_VERIFIED',m.message_id,id);
    });
  }
  health(p:Identity,wire:string) {
    this.ctx.authorise(p,['agent']);
    const m=verifyMessage(wire,{contract:'MC-07',scope:scope(p)},this.ctx.config.trust);
    requireThat(m.contract_id==='MC-07'&&m.body.agent_id===p.agent,'HEALTH_BINDING');
    const b=m.body,time=Date.parse(b.captured_at),verified=Date.parse(b.last_verified_at??'');
    requireThat(b.clock_confidence==='TRUSTED' && time<=Date.now()&&time>Date.now()-300000&&verified<=time && ['site-gateway','site-agent','local-helper','product'].includes(b.component_name) && b.diagnostic_codes.every(c=>['HEALTH_OK','STORAGE_LOW','QUEUE_BACKLOG','LOCAL_OFFLINE'].includes(c)),'HEALTH_POLICY');
    return this.ctx.database.run(p,async tx=>{
      await activeAgent(tx,p);await boundKey(this.ctx,tx,p,m.signature.kid,'agent_telemetry');
      await tx.$executeRaw`SELECT pg_advisory_xact_lock(hashtextextended(${p.agent!},4))`;
      const last=await tx.health_sample.findFirst({where:{agent_id:p.agent,component_code:b.component_name},orderBy:{captured_at:'desc'}});
      requireThat(!last || time>=+last.captured_at+1000,'TELEMETRY_RATE');
      await tx.health_sample.create({data:{...scope(p),id:m.message_id,agent_id:p.agent,component_code:b.component_name,numeric_value:b.queue_depth,captured_at:new Date(time),signed_wire:canon(m)}});
      await audit(tx,p,'HEALTH_INGESTED',m.message_id,p.agent!);
    });
  }
  status(p:Identity,agent:string,product:string) {
    this.ctx.authorise(p,['operator']);uuid(agent);
    return this.ctx.database.run(p,async tx=>{
      const identity=await tx.site_agent.findUniqueOrThrow({where:{id:agent}});
      const sample=await tx.health_sample.findFirst({where:{agent_id:agent},orderBy:{captured_at:'desc'}});
      const ent=await tx.entitlement.findFirst({where:{product_key:product},orderBy:{policy_epoch:'desc'}});
      const last=sample?.signed_wire?JSON.parse(sample.signed_wire).body:null;
      return {identity:identity.status,version:identity.agent_version,health:!last?'UNKNOWN':Date.now()-Date.parse(last.last_verified_at)>300000?'STALE':last.state,lastKnown:last,entitlement:!ent?.signed_wire?'UNKNOWN':+ent.expires_at<=Date.now()?'EXPIRED':Date.parse(JSON.parse(ent.signed_wire).body.not_before)>Date.now()?'NOT_EFFECTIVE':'ASSIGNED',clinicalAuthorisation:'NONE'};
    });
  }
}
