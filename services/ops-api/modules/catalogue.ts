import {randomUUID} from 'node:crypto';
import {verifyMessage} from '../../../packages/signing-ts/index.ts';
import {SyntheticContext, audit, canon, exact, ref, hash, requireThat, scope, uuid, type Identity} from './context.ts';

export class CatalogueService {
  readonly ctx:SyntheticContext;
  constructor(ctx:SyntheticContext) {this.ctx=ctx;}
  publish(p:Identity,wire:string,metadata:{edition:string;dependencies:string[]}={edition:'synthetic',dependencies:[]}) {
    this.ctx.authorise(p,['provisioner']);
    exact(metadata,['edition','dependencies']);ref(metadata.edition);
    requireThat(Array.isArray(metadata.dependencies) && metadata.dependencies.length<=16 && new Set(metadata.dependencies).size===metadata.dependencies.length,'CATALOGUE_METADATA');
    metadata.dependencies.forEach(ref);
    const dependencies=canon([...metadata.dependencies].sort());
    const m=verifyMessage(wire,{contract:'MC-05'},this.ctx.config.trust);
    requireThat(m.contract_id==='MC-05' && Date.parse(m.sent_at)<=Date.now() && Date.parse(m.body.expires_at)>Date.now(),'MANIFEST_EXPIRED');
    requireThat(m.body.health_probes.every(probe=>probe.probe_type==='HTTP' && probe.target==='/local/v1/readiness'),'UNSUPPORTED_PROBE');
    requireThat(m.body.health_probes.length===1,'UNSUPPORTED_PROBE');
    const raw=canon(m), digest=hash(raw);
    return this.ctx.provision.run(p,async tx=>{
      await tx.$executeRaw`SELECT pg_advisory_xact_lock(hashtextextended(${digest},2))`;
      const previous=await tx.deployment_artifact.findUnique({where:{artifact_sha256:digest}});
      if(previous) {requireThat(previous.signed_wire===raw && previous.edition===metadata.edition && previous.service_dependencies===dependencies,'ARTIFACT_CONFLICT');return digest;}
      const id=randomUUID();
      await tx.deployment_artifact.create({data:{id,edition:metadata.edition,service_dependencies:dependencies,product_key:m.body.product_id,artifact_sha256:digest,signed_manifest_sha256:digest,signed_wire:raw,release_id:m.body.release_id,channel:'pilot',state:'PUBLISHED',created_at:new Date()}});
      await audit(tx,p,'ARTIFACT_PUBLISHED',id,id);
      return digest;
    });
  }
  artifact(p:Identity,digest:string) {
    this.ctx.authorise(p,['provisioner','operator']);
    requireThat(/^[a-f0-9]{64}$/.test(digest),'DIGEST');
    return this.ctx.database.run(p,tx=>tx.deployment_artifact.findUniqueOrThrow({where:{artifact_sha256:digest},select:{signed_wire:true,edition:true,service_dependencies:true,channel:true}}));
  }
  entitlement(p:Identity,wire:string) {
    this.ctx.authorise(p,['operator']);
    const m=verifyMessage(wire,{contract:'MC-08',scope:scope(p)},this.ctx.config.trust);
    requireThat(m.contract_id==='MC-08' && Date.parse(m.body.issued_at)<=Date.now() && Date.parse(m.body.expires_at)>Date.parse(m.body.not_before) && m.body.post_grace_mode==='WARN_ONLY','ENTITLEMENT_POLICY');
    const b=m.body;
    return this.ctx.database.run(p,async tx=>{
      await tx.$executeRaw`SELECT pg_advisory_xact_lock(hashtextextended(${p.siteId},3))`;
      const last=await tx.entitlement.findFirst({where:{...scope(p),product_key:b.product_id},orderBy:{policy_epoch:'desc'}});
      requireThat(!last || last.policy_epoch<BigInt(b.policy_epoch),'REVISION_CONFLICT');
      await tx.entitlement.create({data:{...scope(p),id:m.message_id,product_key:b.product_id,policy_epoch:b.policy_epoch,entitlement_sha256:hash(canon(m)),expires_at:new Date(b.expires_at),signed_wire:canon(m)}});
      await audit(tx,p,'ENTITLEMENT_REVISED',m.message_id,b.entitlement_id);
    });
  }
  request(p:Identity,artifact:string,operation:'DEPLOY_RELEASE'|'APPLY_SITE_CONFIG'='DEPLOY_RELEASE') {
    this.ctx.authorise(p,['operator']);
    requireThat(/^[a-f0-9]{64}$/.test(artifact),'DIGEST');
    return this.ctx.database.run(p,async tx=>{
      requireThat(operation==='DEPLOY_RELEASE'||operation==='APPLY_SITE_CONFIG','UNSUPPORTED_OPERATION');
      if(operation==='DEPLOY_RELEASE') await tx.deployment_artifact.findUniqueOrThrow({where:{artifact_sha256:artifact}});
      const id=randomUUID();
      await tx.signing_attestation.create({data:{...scope(p),id,purpose:operation,key_id:p.actor,target_sha256:artifact,created_at:new Date()}});
      await audit(tx,p,'APPROVAL_REQUESTED',id,id);
      return id;
    });
  }
  approve(p:Identity,wire:string) {
    this.ctx.authorise(p,['operator']);
    const m=verifyMessage(wire,{contract:'MC-09',scope:scope(p)},this.ctx.config.trust);
    requireThat(m.contract_id==='MC-09','APPROVAL_CONTRACT');
    const b=m.body, now=Date.now();
    requireThat(b.checker_id===p.actor && ((b.operation==='DEPLOY_RELEASE' && b.purpose==='DEPLOYMENT') || (b.operation==='APPLY_SITE_CONFIG' && b.purpose==='CONFIGURATION_CHANGE')) && Date.parse(b.valid_from)<=now && Date.parse(b.valid_until)>now,'APPROVAL_DENIED');
    return this.ctx.database.run(p,async tx=>{
      const request=await tx.signing_attestation.findFirstOrThrow({where:{id:b.grant_id,purpose:b.operation}});
      requireThat(request.key_id===b.maker_id && b.operator_id===b.maker_id && request.target_sha256===b.target_digest_sha256,'APPROVAL_BINDING');
      const signer=await tx.site_authoriser_key.findFirstOrThrow({where:{id:b.site_authoriser_id,key_id:b.site_authoriser_attestation.kid,revoked_at:null,valid_until:{gt:new Date()}}});
      const key=this.ctx.config.trust.get(signer.key_id);
      requireThat(key && key.alg===signer.algorithm && hash(key.publicKey.export({type:'spki',format:'der'}))===signer.public_key_sha256,'CONSENT_BINDING');
      const resource=b.operation==='DEPLOY_RELEASE'?(await tx.deployment_artifact.findUniqueOrThrow({where:{artifact_sha256:request.target_sha256}})).product_key:'site-config';
      requireThat(b.resource_scope.length===1 && b.resource_scope[0]===resource,'APPROVAL_TARGET');
      await tx.approval_grant.create({data:{...scope(p),id:b.grant_id,target_sha256:b.target_digest_sha256,operation:b.operation,site_authoriser_key_id:b.site_authoriser_attestation.kid,site_authoriser_attestation_sha256:hash(canon(b.site_authoriser_attestation)),ops_grant_sha256:hash(canon(m)),maker_id:b.maker_id,checker_id:b.checker_id,valid_from:new Date(b.valid_from),valid_until:new Date(b.valid_until),signed_wire:canon(m)}});
      await audit(tx,p,'APPROVAL_ISSUED',b.grant_id,request.id);
    });
  }
  revokeGrant(p:Identity,id:string) {
    this.ctx.authorise(p,['operator']);uuid(id);
    return this.ctx.database.run(p,async tx=>{
      await tx.approval_grant.update({where:{id},data:{revoked_at:new Date()}});
      await audit(tx,p,'APPROVAL_REVOKED',id,id);
    });
  }
}
