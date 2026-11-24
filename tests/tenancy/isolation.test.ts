import test,{after,before} from 'node:test';
import assert from 'node:assert/strict';
import {randomUUID} from 'node:crypto';
import {PrismaClient,type Prisma} from '@prisma/client';
import {TenantDatabase} from '../../packages/database/tenant.ts';
import {createOperationsContext} from '../../services/ops-api/app.ts';
const admin=new PrismaClient({datasourceUrl:process.env.TEST_ADMIN_URL}), raw=new PrismaClient();
const wrapper=new TenantDatabase(raw);
const a={tenantId:randomUUID(),siteId:randomUUID()}, b={tenantId:randomUUID(),siteId:randomUUID()}, sibling={tenantId:a.tenantId,siteId:randomUUID()};
const agents=new Map<string,string>(), hash='a'.repeat(64);
before(async()=>{
  assert.equal(new URL(process.env.DATABASE_URL!).username,'zop_runtime');
  for(const s of [a,b,sibling]){
    const install=randomUUID(),agent=randomUUID();agents.set(s.siteId,agent);
    await admin.$executeRaw`INSERT INTO zop.tenant VALUES (${s.tenantId}::uuid,'test','ACTIVE') ON CONFLICT DO NOTHING`;
    await admin.$executeRaw`INSERT INTO zop.site VALUES (${s.siteId}::uuid,${s.tenantId}::uuid,'site-'||${agent},'ACTIVE')`;
    await admin.$executeRaw`INSERT INTO zop.installation VALUES (${install}::uuid,${s.tenantId}::uuid,${s.siteId}::uuid,1,'TEST',${hash})`;
    await admin.$executeRaw`INSERT INTO zop.site_agent VALUES (${agent}::uuid,${s.tenantId}::uuid,${s.siteId}::uuid,${install}::uuid,'1.2.0','TEST')`;
  }
});
after(async()=>{await Promise.all([wrapper.close(),admin.$disconnect()])});
const insertHealth=(tx:Prisma.TransactionClient,s:typeof a)=>tx.$executeRaw`INSERT INTO zop.health_sample VALUES (${randomUUID()}::uuid,${s.tenantId}::uuid,${s.siteId}::uuid,'TEST_OK',1,now())`;
test('migration identity differs from nonowner runtime with FORCE RLS',async()=>{
 const [role]=await raw.$queryRaw<{rolsuper:boolean;rolbypassrls:boolean;rolcreaterole:boolean}[]>`SELECT rolsuper,rolbypassrls,rolcreaterole FROM pg_roles WHERE rolname=current_user`;
 assert.deepEqual(role,{rolsuper:false,rolbypassrls:false,rolcreaterole:false});
 const tables=await admin.$queryRaw<{relname:string;ok:boolean;owner:string}[]>`SELECT relname,relrowsecurity AND relforcerowsecurity AS ok,pg_get_userbyid(relowner) AS owner FROM pg_class WHERE relnamespace='zop'::regnamespace AND relkind='r' AND relname NOT IN ('deployment_artifact','trusted_signing_key','_prisma_migrations')`;
 assert.ok(tables.length>=30 && tables.every(t=>t.ok && t.owner==='zop_migrate'));
});
test('raw SQL without context fails closed on reads and writes',async()=>{
 assert.deepEqual(await raw.$queryRaw`SELECT id FROM zop.site`,[]);
 await assert.rejects(insertHealth(raw,a));
});
test('same-tenant other site and cross-tenant reads and writes',async()=>{
 const sites=await wrapper.run(a,tx=>tx.$queryRaw<{id:string}[]>`SELECT id FROM zop.site`);
 assert.deepEqual(sites,[{id:a.siteId}]);
 for(const wrong of [b,sibling,{tenantId:a.tenantId,siteId:b.siteId}])await assert.rejects(wrapper.run(a,tx=>insertHealth(tx,wrong)));
 await assert.rejects(wrapper.run({tenantId:a.tenantId,siteId:b.siteId},tx=>insertHealth(tx,{tenantId:a.tenantId,siteId:b.siteId})));
});
test('120 concurrent Nest/Prisma transactions across a four-connection pool',async()=>{
 const app=await createOperationsContext();const database=app.get(TenantDatabase);
 try{await Promise.all(Array.from({length:120},(_,i)=>{
   const s=i%2?a:b;
   return database.run(s,async tx=>{
     await insertHealth(tx,s);
     const sites=await tx.site.findMany({select:{id:true,tenant_id:true}});
     assert.deepEqual(sites,[{id:s.siteId,tenant_id:s.tenantId}]);
   });
 }));}finally{await database.close();await app.close()}
 assert.equal(await admin.health_sample.count(),120);
 assert.deepEqual(await raw.site.findMany(),[]);
});
test('foreign keys, idempotency, maker/checker, transitions and atomic single-use consumption',async()=>{
 const agent=agents.get(a.siteId)!,job=randomUUID(),grant=randomUUID();
 const addJob=(id:string,actor:string,seq:number)=>wrapper.run(a,tx=>tx.$executeRaw`INSERT INTO zop.management_job(id,tenant_id,site_id,agent_id,command_type,idempotency_key,nonce,job_sequence,payload_sha256,state,created_at,expires_at,target_sha256) VALUES (${id}::uuid,${a.tenantId}::uuid,${a.siteId}::uuid,${actor}::uuid,'DEPLOY_RELEASE','same',${id},${seq},${hash},'RECEIVED',now(),now()+interval '1 day',${hash})`);
 await assert.rejects(addJob(randomUUID(),agents.get(b.siteId)!,2));await addJob(job,agent,1);await assert.rejects(addJob(randomUUID(),agent,2));
 const addGrant=(checker:string)=>wrapper.run(a,tx=>tx.$executeRaw`INSERT INTO zop.approval_grant VALUES (${grant}::uuid,${a.tenantId}::uuid,${a.siteId}::uuid,${hash},'DEPLOY_RELEASE','test',${hash},${hash},'maker',${checker},now()-interval '1 hour',now()+interval '1 day',NULL)`);
 await assert.rejects(addGrant('maker'));await addGrant('checker');
 const consume=()=>wrapper.run(a,tx=>tx.$executeRaw`INSERT INTO zop.job_approval_grant(id,tenant_id,site_id,job_id,grant_id) VALUES (${randomUUID()}::uuid,${a.tenantId}::uuid,${a.siteId}::uuid,${job}::uuid,${grant}::uuid)`);
 const attempts=await Promise.allSettled([consume(),consume()]);assert.equal(attempts.filter(r=>r.status==='fulfilled').length,1);
 const transition=(s:string)=>wrapper.run(a,tx=>tx.$executeRaw`UPDATE zop.management_job SET state=${s} WHERE id=${job}::uuid`);
 await assert.rejects(transition('SUCCEEDED'));for(const state of ['VALIDATED','ACCEPTED','RUNNING','SUCCEEDED'])await transition(state);
 await assert.rejects(transition('RUNNING'));
});
