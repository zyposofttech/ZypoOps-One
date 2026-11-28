import {killAtReceipt} from './receipt-cut.ts';
import {spawn,execFileSync} from 'node:child_process';
import {createInterface} from 'node:readline';
import {once} from 'node:events';
import {timingSafeEqual} from 'node:crypto';
import assert from 'node:assert/strict';
import {SyntheticContext, canon} from '../../services/ops-api/modules/context.ts';
import {randomUUID} from 'node:crypto';
import {JobService} from '../../services/ops-api/modules/jobs.ts';
import type {createSyntheticApplication} from '../../services/ops-api/modules/application.ts';
import type {fixture} from './fixture.ts';

type App=Awaited<ReturnType<typeof createSyntheticApplication>>;
type F=ReturnType<typeof fixture>;
export async function runProduct(f:F,site:F['installs'][number],plan:string,artifact:string,backupDigest:string,current:()=>App,reopen:()=>Promise<void>) {
  assert.equal(process.env.NODE_ENV,'test');
  const executable=process.env.SYNTHETIC_WORKER!;
  let tool=executable,args=['--synthetic-pipe'];
  if(process.platform==='win32'){
    tool='wsl.exe';
    const distro=process.env.SPRINT3_WSL_DISTRO??'docker-desktop';
    const path=execFileSync(tool,['-d',distro,'-u','nobody','--exec','wslpath','-a',executable],{encoding:'utf8'}).trim();
    args=['-d',distro,'-u','nobody','--exec',path,'--synthetic-pipe'];
  }
  const proc=spawn(tool,args,{stdio:['pipe','pipe','pipe'],windowsHide:true,timeout:60000});
  const completion=once(proc,'close');
  let stderr='';
  proc.stderr.on('data',chunk=>{stderr=(stderr+chunk).slice(-2000);});
  proc.stdin.on('error',()=>{});
  const send=(v:unknown)=>proc.stdin.write(JSON.stringify(v)+'\n');
  const policyKeys=[f.global.job,f.global.release,f.global.approval,site.consent,site.keys.receipt,site.keys.progress,site.keys.backup];
  const privateDer=(kid:string)=>f.priv.get(kid)!.export({type:'pkcs8',format:'der'}).toString('base64');
  send({Identity:{Tenant:site.tenantId,Site:site.siteId,Installation:site.install,Agent:site.agent,HostHash:'a'.repeat(64)},Product:site.product,Artifact:artifact,Job:plan,BackupDigest:backupDigest,Category:site.category,BackupKid:site.keys.backup,ReceiptKid:site.keys.receipt,ReceiptPrivate:privateDer(site.keys.receipt),ProgressKid:site.keys.progress,ProgressPrivate:privateDer(site.keys.progress),Token:site.device.token,Trust:policyKeys.map(kid=>{
    const k=f.trust.get(kid)!;
    return {Kid:kid,Purpose:k.purpose,Alg:k.alg,Tenant:k.tenant_id??'',Site:k.site_id??'',Public:k.publicKey.export({type:'spki',format:'der'}).toString('base64')};
  })});
  let receipts=0,progress=0,done=false,first='';
  try{
    for await(const line of createInterface({input:proc.stdout})){
      assert.ok(Buffer.byteLength(line)<=1048576);
      const req=JSON.parse(line);
      if(req.Op==='done'){
        assert.deepEqual(req.Counts,{effects:1,agentGrants:1,helperGrants:1,acknowledged:1});done=true;break;
      }
      assert.equal(typeof req.Token,'string');
      assert.ok(req.Token.length===64 && timingSafeEqual(Buffer.from(req.Token),Buffer.from(site.device.token)));
      const p=current().get(SyntheticContext).authenticate(req.Token),jobs=current().get(JobService);
      if(req.Op==='next'){
        const replies=await Promise.all([jobs.next(p),jobs.next(p)]);
        const raw=replies.find(Boolean);assert.ok(replies.filter(Boolean).length<=1);
        if(raw) await reopen(); // Persisted dispatch survives teardown before the Agent receives the reply.
        send(raw?{status:'ok',bundle:JSON.parse(raw)}:{status:'empty'});
      }else if(req.Op==='pending'){
        await reopen();send({status:'ok'});
      }else if(req.Op==='progress'){
        const falseReceipt=f.unsigned('MC-04',site,{job_id:plan,execution_id:randomUUID(),observed_sha256:artifact,stage:'VERIFIED',state:'SUCCEEDED',error_code:'NONE',event_at:new Date().toISOString()});
        await assert.rejects(jobs.receipt(p,f.signed(falseReceipt,site.keys.receipt)),/RECEIPT_BINDING/);
        const row=await jobs.deliveryStatus(current().get(SyntheticContext).authenticate(site.maker.token),plan);
        assert.equal(row.receipt,null);assert.equal(row.state,'ACCEPTED');
        await jobs.progress(p,canon(req.Wire));await jobs.progress(p,canon(req.Wire));progress++;
        await reopen();send({status:'ok'});
      }else if(req.Op==='receipt'){
        const wire=canon(req.Wire);
        if(receipts===0 && site===f.installs[0]) {
          const caller=current().get(SyntheticContext).authenticate(site.maker.token);
          const data={token:site.device.token,identity:site.device.identity,kid:site.keys.receipt,publicPem:String(f.trust.get(site.keys.receipt)!.publicKey.export({type:'spki',format:'pem'})),wire};
          await killAtReceipt({...data,cut:'before'});
          assert.equal((await jobs.deliveryStatus(caller,plan)).receipt,null);
          await killAtReceipt({...data,cut:'after'});
          assert.equal((await jobs.deliveryStatus(caller,plan)).receipt,wire);
        }
        await jobs.receipt(p,wire);receipts++;
        if(receipts===1){first=wire;
          if(site===f.installs.at(-1)) f.trust.get(f.global.job)!.revoked=true;
          await reopen();send({status:'lost'});}
        else{assert.equal(wire,first);send({status:'ok'});}
      }else{throw Error('UNSUPPORTED_PIPE_OPERATION');}
    }
    proc.stdin.end();const [code]=await completion;
    assert.equal(code,0,stderr);assert.ok(done);assert.equal(progress,1);assert.equal(receipts,2);
    return first;
  }finally{proc.stdin.end();if(proc.exitCode===null)proc.kill();}
}
