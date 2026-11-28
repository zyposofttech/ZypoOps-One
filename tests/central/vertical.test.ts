import test from 'node:test';
import assert from 'node:assert/strict';
import {randomBytes,randomUUID} from 'node:crypto';
import {setTimeout as delay} from 'node:timers/promises';
import {fixture} from './fixture.ts';
import {runProduct} from './pipe.ts';
import {createSyntheticApplication} from '../../services/ops-api/modules/application.ts';
import {SyntheticContext,canon,hash,scope} from '../../services/ops-api/modules/context.ts';
import {RegistryService} from '../../services/ops-api/modules/registry.ts';
import {EnrolmentService} from '../../services/ops-api/modules/enrolment.ts';
import {CatalogueService} from '../../services/ops-api/modules/catalogue.ts';
import {JobService} from '../../services/ops-api/modules/jobs.ts';
import {EvidenceService} from '../../services/ops-api/modules/evidence.ts';
import {OnboardingService} from '../../services/ops-api/modules/onboarding.ts';
import {ReadinessService} from '../../services/ops-api/modules/readiness.ts';

test('durable scoped management through the enrolled Go Agent and both product profiles',async t=>{
  const f=fixture();
  let app=await createSyntheticApplication(f.config);
  const auth=(s:{token:string})=>app.get(SyntheticContext).authenticate(s.token);
  const reopen=async()=>{await app.close();app=await createSyntheticApplication(f.config);};
  const manifests=new Map<string,string>();
  try{
    for(const [i,s] of f.installs.entries()){
      const reg=app.get(RegistryService),enrol=app.get(EnrolmentService);
      if(i!==1) await reg.createSite(auth(s.provisioner),{code:'simulated-'+i,timezone:'UTC',profile:'SINGLE_SITE_LAN',location:'local',connectivity:'loopback',maintenance:'local'},i===2?undefined:{legalName:'Synthetic Org '+i,details:{contract:'test',tier:'test',contact:'contact-ref',approver:'approver-ref',constraint:'local'}});
      await reg.register(auth(s.provisioner),{id:s.install,hostDigest:'a'.repeat(64),product:s.product});
      await reg.enrolAuthoriser(auth(s.provisioner),s.authoriser,s.consent);
      const claim=await enrol.claim(auth(s.maker),s.install,s.installer.identity.actor);
      const wire=f.enrollment(s,claim);
      if(i===0){
        await t.test('invalid HMAC, CSR, digest, installation, installer and nonce leave the claim unused',async()=>{
          for(const change of [{token_proof:'a'.repeat(43)},{csr_pem:'-----BEGIN CERTIFICATE REQUEST-----\nAAAA\n-----END CERTIFICATE REQUEST-----\n'},{csr_sha256:'b'.repeat(64)},{installation_id:f.installs[1].install},{installer_id:randomUUID()},{claim_nonce:'z'.repeat(32)}]){
            await assert.rejects(async()=>enrol.consume(auth(s.installer),f.enrollment(s,claim,change)));
          }
          const foreign=JSON.parse(f.enrollment(f.installs[1],claim));
          await assert.rejects(enrol.consume(auth(s.installer),f.enrollment(s,claim,{csr_pem:foreign.body.csr_pem,csr_sha256:foreign.body.csr_sha256})),/CSR_BINDING/);
          await assert.rejects(enrol.consume(auth(f.installs[3].installer),wire));
          assert.equal(await app.get(SyntheticContext).database.run(auth(s.maker),tx=>tx.enrolment_claim.count({where:{id:claim.id,consumed_at:null}})),1);
        });
        await t.test('expired bootstrap proof fails without consumption',async()=>{
          const short=await enrol.claim(auth(s.maker),s.install,s.installer.identity.actor,1);
          await delay(1050);
          await assert.rejects(enrol.consume(auth(s.installer),f.enrollment(s,short)),/CLAIM_UNAVAILABLE/);
        });
      }
      const attempts=await Promise.allSettled([enrol.consume(auth(s.installer),wire),enrol.consume(auth(s.installer),wire)]);
      const ok=attempts.find(x=>x.status==='fulfilled');
      if(!ok) throw (attempts[0] as PromiseRejectedResult).reason;
      assert.equal(attempts.filter(r=>r.status==='fulfilled').length,1);
      assert.equal(JSON.parse(ok.value).body.agent_id,s.agent);
      await assert.rejects(enrol.consume(auth(s.installer),wire));
      const competing=await enrol.claim(auth(s.maker),s.install,s.installer.identity.actor);
      await assert.rejects(enrol.consume(auth(s.installer),f.enrollment(s,competing)));
      const cat=app.get(CatalogueService);
      const m=manifests.get(s.product)??f.signed(f.unsigned('MC-05',s,{product_id:s.product,release_id:s.release,required_locks:[s.lock],health_probes:[{name:'app-readiness',probe_type:'HTTP',target:'/local/v1/readiness'}],expires_at:new Date(Date.now()+86400000).toISOString()}),f.global.release);
      manifests.set(s.product,m);
      const digest=await cat.publish(auth(s.provisioner),m);
      assert.equal(await cat.publish(auth(s.provisioner),m),digest);
      await assert.rejects(cat.publish(auth(s.provisioner),m,{edition:'conflicting',dependencies:[]}),/ARTIFACT_CONFLICT/);
      const ent=f.signed(f.unsigned('MC-08',s,{product_id:s.product,issued_at:new Date(Date.now()-2000).toISOString(),not_before:new Date(Date.now()-1000).toISOString(),expires_at:new Date(Date.now()+3600000).toISOString(),policy_epoch:1,offline_grace_days:0}),f.global.entitlement);
      await cat.entitlement(auth(s.maker),ent);
      const bak=randomBytes(32).toString('hex'),input={cpu:'amd64',ramMB:8192,diskGB:100,os:'ubuntu-lts',clockTrusted:true,online:true,profile:'SINGLE_SITE_LAN',backupReady:true};
      const plan=await app.get(ReadinessService).plan(auth(s.maker),s.install,digest,bak,[s.category],input);
      assert.deepEqual(plan.blockers,[]);
      const evidence=app.get(EvidenceService);
      if(i===3) {
        const foreign=auth(f.installs[1].maker),database=app.get(SyntheticContext).database;
        assert.equal(await database.run(foreign,tx=>tx.deployment_plan.count({where:{id:plan.id}})),0);
        await assert.rejects(database.run(foreign,tx=>tx.deployment_plan.create({data:{id:randomUUID(),tenant_id:s.tenantId,site_id:s.siteId,installation_id:s.install,artifact_sha256:digest,backup_digest:randomBytes(32).toString('hex'),categories:[s.category],ready:true,assessment:'{}'}})));
      }
      const manifestTime=new Date().toISOString();
      const backup=f.unsigned('MC-14',s,{backup_manifest_sha256:bak,last_success_at:manifestTime,backup_age_seconds:0,dataset_categories:[s.category]});
      backup.sent_at=manifestTime;
      await assert.rejects(evidence.backup(auth(s.device),plan.id,f.signed({...backup,body:{...backup.body,encrypted:false}},s.keys.backup)),/BACKUP_UNAVAILABLE/);
      if(i===1) {
        const sibling=f.installs[0];
        await assert.rejects(evidence.backup(auth(s.device),plan.id,f.signed(backup,sibling.keys.backup)),/SIGNER_BINDING/);
        await assert.rejects(evidence.backup(auth(sibling.device),plan.id,f.signed(backup,sibling.keys.backup)));
      }
      await evidence.backup(auth(s.device),plan.id,f.signed(backup,s.keys.backup));
      const now=new Date().toISOString();
      const health=f.signed(f.unsigned('MC-07',s,{agent_id:s.agent,captured_at:now,last_verified_at:now}),s.keys.telemetry);
      await evidence.health(auth(s.device),health);
      await assert.rejects(evidence.health(auth(s.device),health));
      await assert.rejects(async()=>evidence.health(auth(s.device),health.replace('"queue_depth"', '"patient_name":"synthetic","queue_depth"')));
      assert.equal((await evidence.status(auth(s.maker),s.agent,s.product)).health,'HEALTHY');
      if(i===0) await t.test('configuration staging binds a dual-signed technical grant and preserves the application gate',async()=>{
        const settings={pollSeconds:60,profile:'LAN',secretRef:'synthetic-ref:gateway'};
        const digest=hash(canon(settings)),id=await cat.request(auth(s.maker),digest,'APPLY_SITE_CONFIG');
        await cat.approve(auth(s.checker),f.grant(s,id,digest,{operation:'APPLY_SITE_CONFIG',purpose:'CONFIGURATION_CHANGE',resource_scope:['site-config']}));
        const ready=app.get(ReadinessService);
        await assert.rejects(ready.stage(auth(s.maker),0,{...settings,secretRef:'unrestricted'},id),/CONFIG_SCHEMA/);
        await assert.rejects(ready.stage(auth(s.maker),3,settings,id),/REVISION_CONFLICT/);
        const revision=await ready.stage(auth(s.maker),0,settings,id);
        assert.equal(revision.status,'STAGED');assert.equal(revision.apply,'JOINT_LOCAL01_APPROVAL_REQUIRED');
        await assert.rejects(ready.stage(auth(s.maker),1,settings,id));
      });
      const grant=await cat.request(auth(s.maker),digest);
      const consent=f.grant(s,grant,digest);
      await assert.rejects(async()=>cat.approve(auth(s.maker),consent));
      if(i===0) {
        const raw=JSON.parse(consent);
        await assert.rejects(cat.approve(auth(s.checker),consent.replace(raw.body.site_authoriser_attestation.value,'A'.repeat(86))));
        await assert.rejects(cat.approve(auth(s.checker),f.grant(s,grant,'0'.repeat(64))));
        await assert.rejects(cat.approve(auth(s.checker),f.grant(s,grant,digest,{valid_until:new Date(Date.now()-1000).toISOString()})));
      }
      await cat.approve(auth(s.checker),consent);
      const idempotency=randomUUID(),jobs=app.get(JobService);
      assert.deepEqual(await Promise.all([jobs.create(auth(s.maker),plan.id,[grant],idempotency),jobs.create(auth(s.maker),plan.id,[grant],idempotency)]),[plan.id,plan.id]);
      await assert.rejects(jobs.create(auth(s.maker),plan.id,[randomUUID()],idempotency),/IDEMPOTENCY_CONFLICT/);
      const other=f.installs[(i+1)%f.installs.length];
      if(i===1 || i===3) assert.equal(await jobs.next(auth(f.installs[0].device)),null);
      else await assert.rejects(jobs.next(auth(other.device)));
      if(i===1) await t.test('unresolved site work blocks a separately authorised competing product before grant consumption',async()=>{
        const other=f.installs[0],otherDigest=hash(manifests.get(other.product)!);
        const competing=await app.get(ReadinessService).plan(auth(other.maker),other.install,otherDigest,randomBytes(32).toString('hex'),[other.category],input);
        const g=await cat.request(auth(other.maker),otherDigest);
        await cat.approve(auth(other.checker),f.grant(other,g,otherDigest));
        await assert.rejects(jobs.create(auth(other.maker),competing.id,[g],randomUUID()),/SITE_HOLD/);
        await app.get(SyntheticContext).database.run(auth(other.maker),async tx=>{
          assert.equal(await tx.management_job.count({where:{id:competing.id}}),0);
          assert.equal((await tx.approval_grant.findUniqueOrThrow({where:{id:g}})).consumed_at,null);
        });
      });
      await reopen();
      await t.test('installation '+(i+1)+' '+s.product+' reconciles the original receipt after central restart and acknowledgement loss',async()=>{
        const wire=await runProduct(f,s,plan.id,digest,bak,()=>app,reopen);
        const terminal=JSON.parse(wire);
        const progress=await app.get(SyntheticContext).database.run(auth(s.maker),tx=>tx.job_progress_event.findFirstOrThrow({where:{job_id:plan.id}}));
        await assert.rejects(app.get(JobService).progress(auth(s.device),progress.signed_wire!));
        const conflict={...terminal,body:{...terminal.body,state:'UNKNOWN',stage:'POST_VERIFY',error_code:'EXECUTION_UNKNOWN'}};
        delete conflict.signature;
        await assert.rejects(app.get(JobService).receipt(auth(s.device),f.signed(conflict,s.keys.receipt)),/OUTCOME_CONFLICT/);
        delete terminal.signature;terminal.body.execution_id=randomUUID();
        await assert.rejects(app.get(JobService).receipt(auth(s.device),f.signed(terminal,s.keys.receipt)),/RECEIPT_BINDING/);
        const p=auth(s.maker);
        await app.get(SyntheticContext).database.run(p,async tx=>{
          const job=await tx.management_job.findUniqueOrThrow({where:{id:plan.id}});
          assert.equal(job.job_sequence,1n);assert.equal(job.state,'SUCCEEDED');
          const receipt=await tx.job_receipt.findUniqueOrThrow({where:{job_id:plan.id}});
          assert.equal(receipt.signed_wire,wire);assert.equal(job.execution_id,JSON.parse(wire).body.execution_id);
          assert.equal(await tx.job_progress_event.count({where:{job_id:plan.id}}),1);
          assert.equal(await tx.job_approval_grant.count({where:{job_id:plan.id}}),1);
          assert.equal(await tx.operational_audit_event.count({where:{...scope(p),correlation_id:plan.id,event_code:'RECEIPT_ACCEPTED'}}),1);
          assert.equal(hash(canon(JSON.parse(wire))),receipt.evidence_sha256);
        });
      });
      const view=await app.get(OnboardingService).assess(auth(s.maker),plan.id);
      assert.equal(view.technical,'VERIFIED_SYNTHETIC');assert.equal(view.clinical,'NO_GO');
      const history=await app.get(OnboardingService).history(auth(s.maker));
      assert.ok(history.events.some(e=>e.outcome==='DENIED'));assert.equal(history.externalAnchor,'UNAVAILABLE');
      if(i===0 || i===2) await t.test(i===0?'expired undelivered command retains its consumed grant':'expired delivered command retains the site hold and has no inferred terminal receipt',async()=>{
        const cat=app.get(CatalogueService),jobs=app.get(JobService),ready=app.get(ReadinessService);
        const bak2=randomBytes(32).toString('hex');
        const plan2=await ready.plan(auth(s.maker),s.install,digest,bak2,[s.category],input);
        const stamp=new Date().toISOString(),backup2=f.unsigned('MC-14',s,{backup_manifest_sha256:bak2,last_success_at:stamp,backup_age_seconds:0,dataset_categories:[s.category]});
        backup2.sent_at=stamp;
        await app.get(EvidenceService).backup(auth(s.device),plan2.id,f.signed(backup2,s.keys.backup));
        await assert.rejects(jobs.create(auth(s.maker),plan2.id,[grant],randomUUID()));
        const grant2=await cat.request(auth(s.maker),digest);
        await cat.approve(auth(s.checker),f.grant(s,grant2,digest,{valid_until:new Date(Date.now()+1500).toISOString()}));
        await jobs.create(auth(s.maker),plan2.id,[grant2],randomUUID());
        if(i===2) assert.ok(await jobs.next(auth(s.device)));
        await delay(1600);
        assert.equal(await jobs.next(auth(s.device)),null);
        const view=await jobs.deliveryStatus(auth(s.maker),plan2.id);
        assert.equal(view.delivery,i===0?'NEVER_DISPATCHED':'RECONCILIATION_REQUIRED');assert.equal(view.receipt,null);
        if(i===2) await assert.rejects(jobs.create(auth(s.maker),plan.id,[grant2],randomUUID()),/SITE_HOLD/);
        await app.get(SyntheticContext).database.run(auth(s.maker),async tx=>{
          assert.equal(await tx.job_approval_grant.count({where:{job_id:plan2.id}}),1);
          assert.equal((await tx.management_job.findUniqueOrThrow({where:{id:plan2.id}})).job_sequence,2n);
        });
      });
      if(i===0) await t.test('expired entitlement reports management expiry and preserves the original execution evidence',async()=>{
        const expired=f.signed(f.unsigned('MC-08',s,{product_id:s.product,issued_at:new Date(Date.now()-120000).toISOString(),not_before:new Date(Date.now()-100000).toISOString(),expires_at:new Date(Date.now()-1000).toISOString(),policy_epoch:2,offline_grace_days:0}),f.global.entitlement);
        await app.get(CatalogueService).entitlement(auth(s.maker),expired);
        const status=await app.get(EvidenceService).status(auth(s.maker),s.agent,s.product);
        assert.equal(status.entitlement,'EXPIRED');
        assert.equal((await app.get(JobService).deliveryStatus(auth(s.maker),plan.id)).state,'SUCCEEDED');
      });
      // Propagate subtest failure to stop subsequent provisioning on an unresolved site hold.
      assert.equal(await app.get(SyntheticContext).database.run(auth(s.maker),tx=>tx.job_receipt.count({where:{job_id:plan.id}})),1);
    }
    const s=f.installs[3];await app.get(EnrolmentService).revoke(auth(s.maker),s.agent);
    await assert.rejects(app.get(JobService).next(auth(s.device)),/IDENTITY_UNAVAILABLE/);
  }finally{await app.close();}
});
