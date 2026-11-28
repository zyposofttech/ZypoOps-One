import {SyntheticContext,canon,hash,scope,uuid,requireThat,type Identity} from './context.ts';

export class OnboardingService {
  readonly ctx:SyntheticContext;
  constructor(ctx:SyntheticContext) {this.ctx=ctx;}
  assess(p:Identity,planId:string) {
    this.ctx.authorise(p,['operator']);uuid(planId);
    return this.ctx.database.run(p,async tx=>{
      const plan=await tx.deployment_plan.findUniqueOrThrow({where:{id:planId}});
      const install=await tx.installation.findUniqueOrThrow({where:{id:plan.installation_id}});
      const agent=await tx.site_agent.findFirst({where:{installation_id:install.id,status:'ENROLLED'}});
      const customer=await tx.tenant.findUniqueOrThrow({where:{id:p.tenantId}});
      const artifact=await tx.deployment_artifact.findUniqueOrThrow({where:{artifact_sha256:plan.artifact_sha256}});
      const ent=await tx.entitlement.findFirst({where:{product_key:artifact.product_key},orderBy:{policy_epoch:'desc'}});
      const job=await tx.management_job.findUnique({where:{id:planId}});
      const receipt=await tx.job_receipt.findUnique({where:{job_id:planId}});
      const checks={customer:customer.status==='ACTIVE' && !!customer.contract_ref && !!customer.approver_ref,site:!!await tx.site.findUnique({where:{id:p.siteId}}),installation:install.product_key===artifact.product_key,agent:!!agent,entitlement:!!ent && +ent.expires_at>Date.now() && !!ent.signed_wire && Date.parse(JSON.parse(ent.signed_wire).body.not_before)<=Date.now(),prerequisites:plan.ready,artifact:!!artifact.signed_wire,backup:!!agent && await tx.backup_status.count({where:{operation_id:planId,agent_id:agent.id,status:'VERIFIED',last_verified_at:{gt:new Date(Date.now()-86400000)}}})>0,approvals:await tx.job_approval_grant.count({where:{job_id:planId}})>0,verifiedOutcome:job?.state==='SUCCEEDED' && receipt?.terminal_state==='SUCCEEDED'};
      return {checks,technical:Object.values(checks).every(Boolean)?'VERIFIED_SYNTHETIC':'INCOMPLETE',clinical:'NO_GO',production:'NO_GO'};
    });
  }
  history(p:Identity,after=0,limit=50) {
    this.ctx.authorise(p,['operator']);
    requireThat(Number.isSafeInteger(after)&&after>=0&&Number.isInteger(limit)&&limit>0&&limit<=100,'PAGE_LIMIT');
    return this.ctx.database.run(p,async tx=>{
      const events=await tx.operational_audit_event.findMany({where:{...scope(p),event_seq:{gt:after}},orderBy:{event_seq:'asc'},take:limit});
      const prior=after?await tx.operational_audit_event.findFirst({where:{...scope(p),event_seq:BigInt(after)}}):null;
      requireThat(!after||prior,'AUDIT_CURSOR');
      let previous=prior?.event_sha256??'0'.repeat(64),seq=BigInt(after);
      for(const row of events){
        const {event_sha256,previous_chain_sha256,created_at,event_seq,...entry}=row;
        requireThat(++seq===event_seq && previous===previous_chain_sha256 && hash(canon({...entry,previous_chain_sha256,created_at:created_at.toISOString(),event_seq:event_seq.toString()}))===event_sha256,'AUDIT_CHAIN');
        previous=event_sha256;
      }
      return {events,cursor:Number(seq),head:previous,externalAnchor:'UNAVAILABLE'};
    });
  }
}
