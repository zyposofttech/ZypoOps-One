import test from 'node:test';
import assert from 'node:assert/strict';
import {randomBytes, randomUUID, generateKeyPairSync} from 'node:crypto';
import {createSyntheticApplication} from '../../services/ops-api/modules/application.ts';
import {SyntheticContext, hash, type Identity} from '../../services/ops-api/modules/context.ts';
import {RegistryService} from '../../services/ops-api/modules/registry.ts';

test('two customers, three sites and four installations retain scope through pooled operations',async()=>{
  const ta=randomUUID(),tb=randomUUID();
  const sites=[{tenantId:ta,siteId:randomUUID()},{tenantId:ta,siteId:randomUUID()},{tenantId:tb,siteId:randomUUID()}];
  const identities=sites.map(s=>({token:randomBytes(32).toString('hex'),identity:{...s,actor:randomUUID(),role:'provisioner' as const}}));
  const unused={kid:'unused',privateKey:generateKeyPairSync('ed25519').privateKey};
  const app=await createSyntheticApplication({identities:identities.map(i=>({tokenHash:hash(i.token),identity:i.identity})),trust:new Map(),targets:new Map(),opsJob:unused,opsIdentity:unused,csrVerifier:'unused',claimKey:randomBytes(32),provisionUrl:process.env.PROVISION_URL!});
  try{
    const ctx=app.get(SyntheticContext),registry=app.get(RegistryService);
    const principals=identities.map(i=>ctx.authenticate(i.token));
    assert.throws(()=>ctx.authenticate('bad'));
    await assert.rejects(()=>registry.inventory({...principals[0],siteId:sites[1].siteId} as Identity));
    const details={contract:'test-contract',tier:'standard',contact:'contact-ref',approver:'approver-ref',constraint:'local-only'};
    for(const [i,p] of principals.entries()){
      await registry.createSite(p,{code:'site-'+i,timezone:'UTC',profile:'lan',location:'site-local',connectivity:'intermittent',maintenance:'local-lock'},i===1?undefined:{legalName:'Synthetic '+i,details});
      await registry.register(p,{id:randomUUID(),hostDigest:'a'.repeat(64),product:'zyposono-sim'});
    }
    const other=await registry.register(principals[0],{id:randomUUID(),hostDigest:'a'.repeat(64),product:'mock-secondary'});
    const views=await Promise.all(principals.map(p=>registry.inventory(p)));
    assert.deepEqual(views.map(v=>v.installations.length),[2,1,1]);
    assert.equal(views[1].entities[0].id,views[0].entities[0].id);
    assert.notEqual(views[2].entities[0].id,views[0].entities[0].id);
    for(const p of principals.slice(1)) await assert.rejects(registry.register(p,{id:randomUUID(),hostDigest:'b'.repeat(64),product:'mock-secondary',replaces:other.id}));
    assert.equal((await registry.inventory(principals[0],1)).installations.length,1);
    await assert.rejects(()=>registry.inventory(principals[0],101));
    await registry.updateCustomer(principals[0],'EXITED',details);
    assert.equal((await registry.inventory(principals[0])).installations.length,2);
  }finally{await app.close();}
});
