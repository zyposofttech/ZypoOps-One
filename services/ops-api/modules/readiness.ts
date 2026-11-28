import {randomUUID} from 'node:crypto';
import {verifyMessage} from '../../../packages/signing-ts/index.ts';
import {SyntheticContext, audit, canon, exact, hash, requireThat, scope, uuid, type Identity} from './context.ts';

export interface Assessment {cpu:string; ramMB:number; diskGB:number; os:string; clockTrusted:boolean; online:boolean; profile:string; backupReady:boolean}
export class ReadinessService {
  readonly ctx:SyntheticContext;
  constructor(ctx:SyntheticContext) {this.ctx=ctx;}
  plan(p:Identity,installationId:string,artifact:string,backupDigest:string,categories:string[],a:Assessment) {
    this.ctx.authorise(p,['operator']);uuid(installationId);
    exact(a,['cpu','ramMB','diskGB','os','clockTrusted','online','profile','backupReady']);
    requireThat([a.cpu,a.os,a.profile].every(x=>typeof x==='string' && /^[A-Za-z0-9_-]{1,40}$/.test(x)) && [a.ramMB,a.diskGB].every(x=>Number.isSafeInteger(x)&&x>=0) && [a.clockTrusted,a.online,a.backupReady].every(x=>typeof x==='boolean'),'ASSESSMENT');
    requireThat(/^[a-f0-9]{64}$/.test(backupDigest) && categories.length>0 && categories.length<=5 && new Set(categories).size===categories.length && categories.every(c=>['APP_CONFIG','MANAGEMENT_AGENT','CLINICAL_DB','DICOM_ARCHIVE','FINANCIAL_DB'].includes(c)),'BACKUP_POLICY');
    return this.ctx.database.run(p,async tx=>{
      const install=await tx.installation.findUniqueOrThrow({where:{id:installationId}});
      const stored=await tx.deployment_artifact.findUniqueOrThrow({where:{artifact_sha256:artifact}});
      const m=verifyMessage(stored.signed_wire??'',{contract:'MC-05'},this.ctx.config.trust);
      requireThat(m.contract_id==='MC-05' && install.product_key===m.body.product_id,'PRODUCT_MISMATCH');
      const b=m.body;
      const checks={cpu:a.cpu===b.cpu_arch,ram:a.ramMB>=b.min_memory_mb,disk:a.diskGB>=b.minimum_disk_gb,os:a.os===b.host_os,clock:a.clockTrusted,profile:a.profile===b.topology,backup:!b.backup_evidence_required||a.backupReady};
      const blockers=Object.entries(checks).filter(([,ok])=>!ok).map(([name])=>name);
      const warnings=a.online?[]:['DISCONNECTED'];
      const row=await tx.deployment_plan.create({data:{...scope(p),id:randomUUID(),installation_id:install.id,artifact_sha256:artifact,backup_digest:backupDigest,categories,assessment:canon({inputs:a,blockers,warnings}),ready:blockers.length===0}});
      await audit(tx,p,'READINESS_ASSESSED',row.id,install.id);
      return {id:row.id,ready:row.ready,blockers,warnings};
    });
  }
  stage(p:Identity,expected:number,settings:{pollSeconds:number;profile:string;secretRef:string},approvalId:string) {
    this.ctx.authorise(p,['operator']);uuid(approvalId);
    exact(settings,['pollSeconds','profile','secretRef']);
    requireThat(Number.isSafeInteger(expected)&&expected>=0 && Number.isInteger(settings.pollSeconds)&&settings.pollSeconds>=15&&settings.pollSeconds<=3600 && ['LAN','INTERMITTENT'].includes(settings.profile) && /^synthetic-ref:[a-z][a-z0-9-]{0,40}$/.test(settings.secretRef),'CONFIG_SCHEMA');
    return this.ctx.database.run(p,async tx=>{
      const raw=canon(settings),digest=hash(raw);
      const g=await tx.approval_grant.findFirstOrThrow({where:{id:approvalId,target_sha256:digest,operation:'APPLY_SITE_CONFIG',revoked_at:null,valid_from:{lte:new Date()},valid_until:{gt:new Date()}}});
      requireThat(g.signed_wire,'CONFIG_APPROVAL');
      const proof=verifyMessage(g.signed_wire,{contract:'MC-09',scope:scope(p)},this.ctx.config.trust);
      requireThat(proof.contract_id==='MC-09' && proof.body.target_digest_sha256===digest,'CONFIG_APPROVAL');
      const prior=await tx.site_configuration_revision.findFirst({orderBy:{revision_no:'desc'}});
      requireThat((prior?.revision_no??0n)===BigInt(expected),'REVISION_CONFLICT');
      const prev=prior?.settings_wire?JSON.parse(prior.settings_wire):{};
      const row=await tx.site_configuration_revision.create({data:{...scope(p),id:randomUUID(),revision_no:expected+1,config_sha256:digest,settings_wire:raw,approval_id:approvalId}});
      await audit(tx,p,'CONFIG_STAGED',row.id,approvalId);
      return {revision:expected+1,digest,changed:Object.keys(settings).filter(k=>prev[k]!==settings[k as keyof typeof settings]),status:'STAGED',apply:'JOINT_LOCAL01_APPROVAL_REQUIRED'};
    });
  }
}
