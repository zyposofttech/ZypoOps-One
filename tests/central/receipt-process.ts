import {createPublicKey, generateKeyPairSync, randomBytes} from 'node:crypto';
import {createSyntheticApplication} from '../../services/ops-api/modules/application.ts';
import {SyntheticContext, hash, type Identity} from '../../services/ops-api/modules/context.ts';
import {JobService} from '../../services/ops-api/modules/jobs.ts';
import type {TrustedKey} from '../../packages/signing-ts/index.ts';

export interface ReceiptCut {
  cut:'before'|'after'; token:string;identity:Identity;kid:string;publicPem:string;wire:string;
}
async function pause(cut:string):Promise<never> {
  process.send?.({barrier:cut});
  // The parent terminates this dedicated process while its Prisma transaction is at the boundary.
  return new Promise(()=>{setInterval(()=>{},1000);});
}
async function execute(input:ReceiptCut) {
  if(process.env.NODE_ENV!=='test' || !process.send) throw Error('TEST_ONLY');
  const {identity,kid}=input;
  const key:TrustedKey={purpose:'agent_receipt',alg:'Ed25519',publicKey:createPublicKey(input.publicPem),tenant_id:identity.tenantId,site_id:identity.siteId};
  const unused={kid:'unused',privateKey:generateKeyPairSync('ed25519').privateKey};
  const app=await createSyntheticApplication({identities:[{tokenHash:hash(input.token),identity}],trust:new Map([[kid,key]]),targets:new Map(),opsIdentity:unused,opsJob:unused,claimKey:randomBytes(32),csrVerifier:'unused',provisionUrl:process.env.PROVISION_URL!});
  const ctx=app.get(SyntheticContext);
  if(input.cut==='before') {
    const run=ctx.database.run.bind(ctx.database);
    ctx.database.run=(scope,operation)=>run(scope,async tx=>{
      await operation(tx);
      return pause('before');
    });
  }
  await app.get(JobService).receipt(ctx.authenticate(input.token),input.wire);
  await pause('after');
}
if(process.send && process.env.NODE_ENV==='test') process.once('message',input=>{
  execute(input as ReceiptCut).catch(()=>{process.send?.({error:'RECEIPT_CHILD_FAILED'});process.exitCode=1;process.disconnect();});
});
