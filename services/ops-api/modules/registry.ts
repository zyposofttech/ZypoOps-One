import {randomUUID} from 'node:crypto';
import {SyntheticContext, audit, exact, hash, ref, requireThat, scope, uuid, type Identity} from './context.ts';

export interface CustomerDetails {contract: string; tier: string; contact: string; approver: string; constraint: string}
export interface SiteDetails {code: string; timezone: string; profile: string; location: string; connectivity: string; maintenance: string}

export class RegistryService {
  readonly ctx:SyntheticContext;
  constructor(ctx:SyntheticContext) {this.ctx=ctx;}
  createSite(p:Identity,site:SiteDetails,customer?:{legalName:string;details:CustomerDetails}) {
    this.ctx.authorise(p,['provisioner']);
    exact(site,['code','timezone','profile','location','connectivity','maintenance']);
    Object.values(site).forEach(ref);
    new Intl.DateTimeFormat('en',{timeZone:site.timezone});
    const attrs={timezone:site.timezone,deployment_profile:site.profile,data_location:site.location,connectivity:site.connectivity,maintenance:site.maintenance};
    return this.ctx.provision.run(p,async tx=>{
      if(customer) {
        requireThat(typeof customer.legalName==='string' && /^[A-Za-z0-9 .&-]{1,160}$/.test(customer.legalName),'INVALID_LEGAL_NAME');
        await tx.tenant.create({data:{id:p.tenantId,legal_name:customer.legalName,status:'ACTIVE',...this.customerFields(customer.details)}});
        await tx.customer_legal_entity.create({data:{id:randomUUID(),tenant_id:p.tenantId,legal_name:customer.legalName}});
      }
      await tx.site.create({data:{id:p.siteId,tenant_id:p.tenantId,site_code:site.code,status:'ACTIVE',...attrs}});
      if(customer) await audit(tx,p,'CUSTOMER_CREATED',p.tenantId,p.siteId);
      await audit(tx,p,'SITE_PROVISIONED',p.siteId,p.siteId);
      return p.siteId;
    });
  }
  private customerFields(d:CustomerDetails) {
    exact(d,['contract','tier','contact','approver','constraint']);
    Object.values(d).forEach(ref);
    return {contract_ref:d.contract,support_tier:d.tier,contact_ref:d.contact,approver_ref:d.approver,deployment_constraint:d.constraint};
  }
  updateCustomer(p:Identity,status:'ACTIVE'|'SUSPENDED_FOR_ONBOARDING'|'EXITED',details:CustomerDetails) {
    this.ctx.authorise(p,['provisioner']);
    requireThat(['ACTIVE','SUSPENDED_FOR_ONBOARDING','EXITED'].includes(status),'LIFECYCLE');
    return this.ctx.provision.run(p,async tx=>{
      const customer=await tx.tenant.update({where:{id:p.tenantId},data:{status,...this.customerFields(details)}});
      await audit(tx,p,'CUSTOMER_UPDATED',p.tenantId,randomUUID());
      return customer;
    });
  }
  register(p:Identity,input:{id:string;hostDigest:string;product:string;replaces?:string}) {
    this.ctx.authorise(p,['provisioner']);
    exact(input,input.replaces?['id','hostDigest','product','replaces']:['id','hostDigest','product']);
    uuid(input.id);ref(input.product);
    requireThat(/^[a-f0-9]{64}$/.test(input.hostDigest),'HOST_BINDING');
    if(input.replaces) uuid(input.replaces);
    return this.ctx.provision.run(p,async tx=>{
      let epoch=1;
      if(input.replaces) {
        const prior=await tx.installation.findUniqueOrThrow({where:{id:input.replaces}});
        epoch=prior.installation_epoch+1;
      }
      const row=await tx.installation.create({data:{...scope(p),id:input.id,product_key:input.product,host_binding_sha256:input.hostDigest,replaces_id:input.replaces,installation_epoch:epoch,status:'REGISTERED'}});
      await audit(tx,p,'INSTALLATION_REGISTERED',row.id,randomUUID());
      return row;
    });
  }
  transferOwnership(p:Identity,siteId:string) {
    this.ctx.authorise(p,['provisioner']);uuid(siteId);
    requireThat(siteId===p.siteId,'FORBIDDEN');
    return {siteId,status:'SIGNED_TRANSFER_CEREMONY_REQUIRED'} as const;
  }
  enrolAuthoriser(p:Identity,id:string,kid:string) {
    this.ctx.authorise(p,['provisioner']);uuid(id);
    const k=this.ctx.config.trust.get(kid);
    requireThat(k && !k.revoked && k.purpose==='site_authoriser' && k.tenant_id===p.tenantId && k.site_id===p.siteId,'CONSENT_BINDING');
    return this.ctx.provision.run(p,async tx=>{
      await tx.site_authoriser_key.create({data:{...scope(p),id,key_id:kid,algorithm:k.alg,public_key_sha256:hash(k.publicKey.export({type:'spki',format:'der'})),valid_until:new Date(Date.now()+86400000)}});
      await audit(tx,p,'SITE_AUTHORISER_ENROLLED',id,id);
    });
  }
  inventory(p:Identity,limit=20,after?:string) {
    this.ctx.authorise(p,['operator','provisioner']);
    requireThat(Number.isInteger(limit)&&limit>=1&&limit<=100,'PAGE_LIMIT');
    if(after) uuid(after);
    return this.ctx.database.run(p,async tx=>({
      customer:await tx.tenant.findUniqueOrThrow({where:{id:p.tenantId}}),
      entities:await tx.customer_legal_entity.findMany({take:100,orderBy:{id:'asc'}}),
      site:await tx.site.findUniqueOrThrow({where:{id:p.siteId}}),
      installations:await tx.installation.findMany({where:{...scope(p),id:{gt:after}},take:limit,orderBy:{id:'asc'}})
    }));
  }
}
