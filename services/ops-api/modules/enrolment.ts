import {createHmac, createPublicKey, randomBytes, randomUUID, timingSafeEqual} from 'node:crypto';
import {execFile} from 'node:child_process';
import {decode} from '../../../packages/management-contracts/index.ts';
import {signMessage, verifyMessage} from '../../../packages/signing-ts/index.ts';
import {SyntheticContext, audit, canon, hash, requireThat, scope, uuid, type Identity} from './context.ts';

function checkedCSR(executable:string,pem:string):Promise<{digest:string;publicKey:string}> {
  return new Promise((resolve,reject)=>{
    const proc=execFile(executable,[],{timeout:2000,maxBuffer:8192,windowsHide:true},(err,stdout)=>{
      if(err) {reject(Error('INVALID_CSR'));return;}
      try { resolve(JSON.parse(stdout)); } catch { reject(Error('INVALID_CSR')); }
    });
    proc.stdin?.on('error',()=>{});
    proc.stdin?.end(pem);
  });
}
export class EnrolmentService {
  readonly ctx:SyntheticContext;
  constructor(ctx:SyntheticContext) {this.ctx=ctx;}
  private secret(id:string,nonce:string) {
    return createHmac('sha256',this.ctx.config.claimKey).update('synthetic-bootstrap\0'+id+'\0'+nonce).digest();
  }
  claim(p:Identity,installation:string,installer:string,ttlSeconds=300) {
    this.ctx.authorise(p,['operator']);uuid(installation);uuid(installer);
    requireThat(Number.isInteger(ttlSeconds)&&ttlSeconds>=1&&ttlSeconds<=600,'CLAIM_TTL');
    const id=randomUUID(), nonce=randomBytes(24).toString('base64url'), secret=this.secret(id,nonce);
    return this.ctx.database.run(p,async tx=>{
      await tx.installation.findUniqueOrThrow({where:{id:installation}});
      const issued=new Date(), expiry=new Date(+issued+ttlSeconds*1000);
      await tx.enrolment_claim.create({data:{...scope(p),id,installation_id:installation,installer_id:installer,claim_nonce:nonce,token_hash:hash(secret),issued_at:issued,expires_at:expiry}});
      await audit(tx,p,'CLAIM_CREATED',id,installation);
      return {id,nonce,bootstrapSecret:secret.toString('hex'),expiresAt:expiry};
    });
  }
  async consume(p:Identity,wire:string) {
    this.ctx.authorise(p,['installer']);
    const doc=decode(wire);
    requireThat(doc.contract_id==='MC-02','ENROLMENT_CONTRACT');
    const b=doc.body, cfg=this.ctx.config, target=cfg.targets.get(b.installation_id);
    requireThat(target && p.actor===b.installer_id && doc.signature.kid===target.pop && b.key_algorithm==='Ed25519','ENROLMENT_BINDING');
    verifyMessage(wire,{contract:'MC-02',scope:scope(p)},cfg.trust);
    const csr=await checkedCSR(cfg.csrVerifier,b.csr_pem);
    requireThat(csr.digest===b.csr_sha256 && createPublicKey(csr.publicKey).equals(cfg.trust.get(target.pop)!.publicKey),'CSR_BINDING');
    const secret=this.secret(b.claim_id,b.claim_nonce);
    const mac=createHmac('sha256',secret).update(canon({claim_id:b.claim_id,installation_id:b.installation_id,claim_nonce:b.claim_nonce,csr_sha256:b.csr_sha256,site_id:p.siteId})).digest('base64url');
    requireThat(Buffer.byteLength(mac)===Buffer.byteLength(b.token_proof) && timingSafeEqual(Buffer.from(mac),Buffer.from(b.token_proof)),'TOKEN_PROOF');
    return this.ctx.database.run(p,async tx=>{
      const install=await tx.installation.findUniqueOrThrow({where:{id:b.installation_id}});
      requireThat(install.host_binding_sha256===b.host_evidence_sha256,'HOST_BINDING');
      const now=new Date();
      const claim=await tx.enrolment_claim.updateMany({where:{...scope(p),id:b.claim_id,installation_id:install.id,installer_id:p.actor,claim_nonce:b.claim_nonce,token_hash:hash(secret),consumed_at:null,expires_at:{gt:now}},data:{consumed_at:now,consumed_csr_sha256:csr.digest}});
      requireThat(claim.count===1,'CLAIM_UNAVAILABLE');
      const serial=randomUUID();
      const identity=signMessage(canon({contract_id:'MC-01',schema_version:'1.2.0',message_id:randomUUID(),sent_at:now.toISOString(),...scope(p),body:{installation_id:install.id,agent_id:target.agent,host_binding_sha256:install.host_binding_sha256,cert_serial:serial,key_version:'synthetic-v1',capabilities:['HEALTH_MONITORING'],status:'ENROLLED',installation_epoch:install.installation_epoch}}),{contract:'MC-01',scope:scope(p)},cfg.trust,cfg.opsIdentity.kid,cfg.opsIdentity.privateKey);
      await tx.site_agent.create({data:{...scope(p),id:target.agent,installation_id:install.id,receipt_kid:target.receipt,identity_wire:identity,agent_version:b.agent_version,status:'ENROLLED'}});
      const until=new Date(+now+24*3600000);
      await tx.certificate_identity.create({data:{...scope(p),id:randomUUID(),installation_id:install.id,issuer:'SYNTHETIC-IDENTITY',serial,subject_sha256:csr.digest,not_before:now,not_after:until}});
      for(const [kid,purpose] of [[target.receipt,'agent_receipt'],[target.telemetry,'agent_telemetry'],[target.backup,'agent_backup_receipt'],[target.progress,'agent_progress']]) {
        const key=cfg.trust.get(kid);
        requireThat(key && !key.revoked && key.purpose===purpose && key.tenant_id===p.tenantId && key.site_id===p.siteId,'ENROLMENT_KEY');
        await tx.site_signing_key.create({data:{...scope(p),kid,agent_id:target.agent,purpose,algorithm:key.alg,public_key_pem:String(key.publicKey.export({type:'spki',format:'pem'})),root_revision:1n,valid_from:now,valid_until:until}});
      }
      await audit(tx,p,'CLAIM_CONSUMED',b.claim_id,install.id);
      await audit(tx,p,'IDENTITY_ENROLLED',target.agent,install.id);
      return identity;
    });
  }
  revoke(p:Identity,agentId:string) {
    this.ctx.authorise(p,['operator']);uuid(agentId);
    return this.ctx.database.run(p,async tx=>{
      const agent=await tx.site_agent.findFirstOrThrow({where:{id:agentId,status:'ENROLLED'}});
      const now=new Date();
      const cert=await tx.certificate_identity.findFirstOrThrow({where:{installation_id:agent.installation_id,revoked_at:null}});
      const changed=await tx.site_agent.updateMany({where:{id:agentId,status:'ENROLLED'},data:{status:'REVOKED'}});
      requireThat(changed.count===1,'IDENTITY_UNAVAILABLE');
      await tx.site_signing_key.updateMany({where:{agent_id:agentId},data:{revoked_at:now}});
      await tx.certificate_identity.update({where:{id:cert.id},data:{revoked_at:now}});
      await tx.revocation_record.create({data:{...scope(p),id:randomUUID(),certificate_id:cert.id,reason_code:'SYNTHETIC_REVOCATION',revoked_at:now,revocation_epoch:1n}});
      await audit(tx,p,'IDENTITY_REVOKED',agentId,cert.id);
    });
  }
}
