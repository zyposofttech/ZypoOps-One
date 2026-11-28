import {createHash, randomBytes, randomUUID} from 'node:crypto';
import {signMessage,verifyMessage} from '../../../packages/signing-ts/index.ts';
import {SyntheticContext, activeAgent, audit, canon, hash, requireThat, scope, uuid, type Identity, type Tx} from './context.ts';
import {boundKey} from './evidence.ts';

function executionId(binding:string) {
  // uuid.NewSHA1(uuid.NameSpaceOID, []byte(binding)) in packages/deployment.
  const id=createHash('sha1').update(Buffer.from('6ba7b8129dad11d180b400c04fd430c8','hex')).update(binding).digest().subarray(0,16);
  id[6]=(id[6]&15)|80;id[8]=(id[8]&63)|128;
  return id.toString('hex').replace(/^(.{8})(.{4})(.{4})(.{4})(.{12})$/,'$1-$2-$3-$4-$5');
}
async function siteLock(tx:Tx,site:string) {
  await tx.$executeRaw`SELECT pg_advisory_xact_lock(hashtextextended(${site},5))`;
}
const acceptedOutcomes=new Set(['VERIFIED/SUCCEEDED/NONE','BEFORE_EFFECT/REJECTED/PRECONDITION_FAILED','POST_VERIFY/FAILED/PRECONDITION_FAILED','POST_VERIFY/UNKNOWN/EXECUTION_UNKNOWN','AFTER_EFFECT/FAILED/OTHER_CODED']);
export class JobService {
  readonly ctx:SyntheticContext;
  constructor(ctx:SyntheticContext) {this.ctx=ctx;}
  create(p:Identity,planId:string,grantIds:string[],idempotency:string) {
    this.ctx.authorise(p,['operator']);uuid(planId);uuid(idempotency);
    requireThat(Array.isArray(grantIds)&&grantIds.length>0&&grantIds.length<=8&&new Set(grantIds).size===grantIds.length,'APPROVAL_LIST');grantIds.forEach(uuid);
    const request=hash(canon({planId,grants:[...grantIds].sort()}));
    const cfg=this.ctx.config;
    return this.ctx.database.run(p,async tx=>{
      await siteLock(tx,p.siteId);
      const old=await tx.management_job.findFirst({where:{...scope(p),command_type:'DEPLOY_RELEASE',idempotency_key:idempotency}});
      if(old) {requireThat(old.request_sha256===request,'IDEMPOTENCY_CONFLICT');return old.id;}
      requireThat(await tx.tenant.count({where:{id:p.tenantId,status:'ACTIVE'}})===1,'CUSTOMER_INACTIVE');
      const plan=await tx.deployment_plan.findUniqueOrThrow({where:{id:planId}});
      requireThat(plan.ready,'PREREQUISITE_FAILED');
      const hold=await tx.management_job.findFirst({where:{...scope(p),command_type:'DEPLOY_RELEASE',state:{notIn:['SUCCEEDED','REJECTED','CANCELLED']}}});
      requireThat(!hold,'SITE_HOLD');
      const agent=await tx.site_agent.findFirstOrThrow({where:{installation_id:plan.installation_id,status:'ENROLLED'}});
      const asAgent={...p,role:'agent' as const,agent:agent.id,installation:agent.installation_id};
      const artifact=await tx.deployment_artifact.findUniqueOrThrow({where:{artifact_sha256:plan.artifact_sha256}});
      const manifest=verifyMessage(artifact.signed_wire??'',{contract:'MC-05'},cfg.trust);
      requireThat(manifest.contract_id==='MC-05'&&Date.parse(manifest.body.expires_at)>Date.now(),'ARTIFACT_EXPIRED');
      const parts=(v:string)=>v.split('.').map(Number);
      const av=parts(agent.agent_version),min=parts(manifest.body.min_agent_version);
      requireThat(av.length===3 && min.length===3 && [...av,...min].every(Number.isSafeInteger) && (av[0]>min[0]||av[0]===min[0]&&(av[1]>min[1]||av[1]===min[1]&&av[2]>=min[2])),'AGENT_VERSION');
      const ent=await tx.entitlement.findFirstOrThrow({where:{product_key:artifact.product_key},orderBy:{policy_epoch:'desc'}});
      const entitlement=verifyMessage(ent.signed_wire??'',{contract:'MC-08',scope:scope(p)},cfg.trust);
      requireThat(entitlement.contract_id==='MC-08' && Date.parse(entitlement.body.not_before)<=Date.now() && +ent.expires_at>Date.now(),'ENTITLEMENT_INACTIVE');
      const backupRow=await tx.backup_status.findFirstOrThrow({where:{operation_id:plan.id,agent_id:agent.id,target_sha256:plan.artifact_sha256},orderBy:{last_verified_at:'desc'}});
      const backup=verifyMessage(backupRow.signed_wire??'',{contract:'MC-14',scope:scope(p)},cfg.trust);
      requireThat(backup.contract_id==='MC-14' && backupRow.last_verified_at && +backupRow.last_verified_at+86400000>Date.now(),'BACKUP_STALE');
      await boundKey(this.ctx,tx,asAgent,backup.signature.kid,'agent_backup_receipt');
      const proofs=[];
      let deadline=Math.min(Date.now()+300000,Date.parse(manifest.body.expires_at),+ent.expires_at,+backupRow.last_verified_at+86400000);
      for(const id of grantIds) {
        await tx.$queryRaw`SELECT id FROM zop.approval_grant WHERE id=${id}::uuid FOR UPDATE`;
        const grant=await tx.approval_grant.findFirstOrThrow({where:{id,revoked_at:null,consumed_at:null,target_sha256:plan.artifact_sha256,operation:'DEPLOY_RELEASE'}});
        const proof=verifyMessage(grant.signed_wire??'',{contract:'MC-09',scope:scope(p)},cfg.trust);
        requireThat(proof.contract_id==='MC-09' && proof.body.resource_scope.includes(artifact.product_key) && +grant.valid_from<=Date.now(),'GRANT_BINDING');
        const siteKey=await tx.site_authoriser_key.count({where:{id:proof.body.site_authoriser_id,key_id:proof.body.site_authoriser_attestation.kid,revoked_at:null,valid_until:{gt:new Date()}}});
        requireThat(siteKey===1,'CONSENT_BINDING');
        deadline=Math.min(deadline,+grant.valid_until);proofs.push(proof);
      }
      const now=new Date();requireThat(deadline>+now,'AUTHORITY_EXPIRED');
      const seq=(await tx.management_job.aggregate({where:{installation_id:plan.installation_id},_max:{job_sequence:true}}))._max.job_sequence??0n;
      const sequence=Number(seq+1n);requireThat(Number.isSafeInteger(sequence),'SEQUENCE_LIMIT');
      const nonce=randomBytes(24).toString('base64url');
      const body={job_id:plan.id,installation_id:plan.installation_id,agent_id:agent.id,command_type:'DEPLOY_RELEASE',params:{manifest_sha256:plan.artifact_sha256,product_key:artifact.product_key,release_channel:'pilot'},sequence,nonce,idempotency_key:idempotency,issued_at:now.toISOString(),not_before:now.toISOString(),expires_at:new Date(deadline).toISOString(),min_agent_version:manifest.body.min_agent_version,approval_grant_ids:grantIds,preconditions:{}};
      const signed=signMessage(canon({contract_id:'MC-03',schema_version:'1.2.0',message_id:randomUUID(),sent_at:now.toISOString(),...scope(p),body}),{contract:'MC-03',scope:scope(p)},cfg.trust,cfg.opsJob.kid,cfg.opsJob.privateKey);
      const bundle=canon({job:JSON.parse(signed),manifest,grants:proofs,backup});
      await tx.management_job.create({data:{...scope(p),id:plan.id,agent_id:agent.id,installation_id:plan.installation_id,command_type:'DEPLOY_RELEASE',idempotency_key:idempotency,nonce,job_sequence:sequence,payload_sha256:hash(signed),state:'RECEIVED',created_at:now,expires_at:new Date(deadline),target_sha256:plan.artifact_sha256,bundle_wire:bundle,execution_id:executionId(hash(bundle)),request_sha256:request}});
      for(const grant_id of grantIds) await tx.job_approval_grant.create({data:{...scope(p),id:randomUUID(),job_id:plan.id,grant_id}});
      for(const state of ['VALIDATED','ACCEPTED']) await tx.management_job.update({where:{id:plan.id},data:{state}});
      await tx.outbox_event.create({data:{...scope(p),id:randomUUID(),event_type:'COMMAND_AVAILABLE',body_sha256:hash(bundle),created_at:now}});
      await audit(tx,p,'JOB_CREATED',plan.id,plan.id);
      return plan.id;
    });
  }
  next(p:Identity) {
    this.ctx.authorise(p,['agent']);
    return this.ctx.database.run(p,async tx=>{
      await siteLock(tx,p.siteId);await activeAgent(tx,p);
      const job=await tx.management_job.findFirst({where:{agent_id:p.agent,installation_id:p.installation,state:'ACCEPTED'},orderBy:{job_sequence:'asc'}});
      if(!job) return null;
      const prior=await tx.job_attempt.findFirst({where:{job_id:job.id},orderBy:{attempt_no:'desc'}});
      const now=new Date();
      const grants=await tx.job_approval_grant.findMany({where:{job_id:job.id},include:{approval_grant:true}});
      const bundle=JSON.parse(job.bundle_wire!);
      const key=this.ctx.config.trust.get(bundle.job.signature.kid);
      const revokedProof = [bundle.manifest.signature.kid,bundle.backup.signature.kid,...bundle.grants.flatMap((g:{signature:{kid:string};body:{site_authoriser_attestation:{kid:string}}})=>[g.signature.kid,g.body.site_authoriser_attestation.kid])].some(kid=>{
        const proof=this.ctx.config.trust.get(kid);
        return !proof || proof.revoked;
      });
      if(+job.expires_at<=+now || !key || key.revoked || revokedProof || grants.some(g=>g.approval_grant.revoked_at!==null)) {
        if(!prior) {
          await tx.management_job.update({where:{id:job.id},data:{state:'REJECTED'}});
          await audit(tx,p,'UNDISPATCHED_AUTHORITY_ENDED',job.id,job.id);
        }
        return null;
      }
      if(prior && (+now-+prior.started_at<250 || prior.attempt_no>=20)) return null;
      await tx.job_attempt.create({data:{...scope(p),id:randomUUID(),job_id:job.id,attempt_no:(prior?.attempt_no??0)+1,state:'DELIVERED',started_at:now}});
      await audit(tx,p,'COMMAND_DISPATCHED',job.id,job.id);
      return job.bundle_wire;
    });
  }
  deliveryStatus(p:Identity,id:string) {
    this.ctx.authorise(p,['operator']);uuid(id);
    return this.ctx.database.run(p,async tx=>{
      const job=await tx.management_job.findUniqueOrThrow({where:{id}});
      const attempts=await tx.job_attempt.count({where:{job_id:id}});
      const receipt=await tx.job_receipt.findUnique({where:{job_id:id}});
      const unresolved=job.state==='ACCEPTED' && attempts>0 && (+job.expires_at<=Date.now() || attempts>=20);
      return {id,state:job.state,attempts,delivery:receipt?'TERMINAL_RECEIPT':unresolved?'RECONCILIATION_REQUIRED':attempts?'AWAITING_RECEIPT':job.state==='REJECTED'?'NEVER_DISPATCHED':'QUEUED',receipt:receipt?.signed_wire??null};
    });
  }
  progress(p:Identity,wire:string) {
    this.ctx.authorise(p,['agent']);
    const m=verifyMessage(wire,{contract:'AUX-01',scope:scope(p)},this.ctx.config.trust);
    requireThat(m.contract_id==='AUX-01','PROGRESS_CONTRACT');
    return this.ctx.database.run(p,async tx=>{
      await siteLock(tx,p.siteId);await activeAgent(tx,p);await boundKey(this.ctx,tx,p,m.signature.kid,'agent_progress');
      const b=m.body,job=await tx.management_job.findFirstOrThrow({where:{id:b.job_id,agent_id:p.agent,installation_id:p.installation,state:'ACCEPTED'}});
      requireThat(b.execution_id===job.execution_id && /^(VERIFY_ARTIFACT|PREFLIGHT|POST_VERIFY|BEFORE_EFFECT|VERIFIED)$/.test(b.stage_code),'PROGRESS_BINDING');
      const prev=await tx.job_progress_event.findFirst({where:{job_id:job.id},orderBy:{sequence_number:'desc'}});
      if(prev?.sequence_number===BigInt(b.sequence_number) && prev.signed_wire===canon(m)) return;
      const states=['RECEIVED','VALIDATED','ACCEPTED','RUNNING'];
      requireThat(BigInt(b.sequence_number)===(prev?.sequence_number??0n)+1n && (!prev||states.indexOf(b.progress_state)>=states.indexOf(prev.progress_state)),'PROGRESS_SEQUENCE');
      await tx.job_progress_event.create({data:{...scope(p),id:m.message_id,job_id:job.id,stage_code:b.stage_code,progress_state:b.progress_state,event_at:new Date(b.event_at),sequence_number:b.sequence_number,signed_wire:canon(m)}});
      await audit(tx,p,'PROGRESS_ACCEPTED',m.message_id,job.id);
    });
  }
  receipt(p:Identity,wire:string) {
    this.ctx.authorise(p,['agent']);
    const m=verifyMessage(wire,{contract:'MC-04',scope:scope(p)},this.ctx.config.trust);
    requireThat(m.contract_id==='MC-04','RECEIPT_CONTRACT');
    return this.ctx.database.run(p,async tx=>{
      await siteLock(tx,p.siteId);
      const identity=await activeAgent(tx,p);await boundKey(this.ctx,tx,p,m.signature.kid,'agent_receipt');
      const b=m.body,job=await tx.management_job.findFirstOrThrow({where:{id:b.job_id,agent_id:p.agent,installation_id:p.installation}});
      requireThat(identity.receipt_kid===m.signature.kid && job.execution_id===b.execution_id && job.target_sha256===b.observed_sha256 && b.clock_confidence==='TRUSTED' && acceptedOutcomes.has([b.stage,b.state,b.error_code].join('/')),'RECEIPT_BINDING');
      const prior=await tx.job_receipt.findUnique({where:{job_id:job.id}});
      if(prior) { requireThat(prior.evidence_sha256===hash(canon(m)),'OUTCOME_CONFLICT');return; }
      requireThat(job.state==='ACCEPTED' && await tx.job_attempt.count({where:{job_id:job.id}})>0,'UNDELIVERED');
      await tx.job_receipt.create({data:{...scope(p),id:m.message_id,job_id:job.id,terminal_state:b.state,evidence_sha256:hash(canon(m)),completed_at:new Date(b.event_at),signed_wire:wire}});
      if(b.state!=='REJECTED') await tx.management_job.update({where:{id:job.id},data:{state:'RUNNING'}});
      await tx.management_job.update({where:{id:job.id},data:{state:b.state}});
      await audit(tx,p,'RECEIPT_ACCEPTED',m.message_id,job.id);
    });
  }
}
