import {fork} from 'node:child_process';
import assert from 'node:assert/strict';
import type {ReceiptCut} from './receipt-process.ts';

export async function killAtReceipt(input:ReceiptCut) {
  const child=fork(new URL('./receipt-process.ts',import.meta.url),[],{stdio:['ignore','ignore','ignore','ipc'],execArgv:[]});
  const exit=new Promise<void>(resolve=>child.once('exit',()=>resolve()));
  try {
    await new Promise<void>((resolve,reject)=>{
      const timeout=setTimeout(()=>reject(Error('RECEIPT_BARRIER_TIMEOUT')),10000);
      child.once('error',err=>{clearTimeout(timeout);reject(err);});
      child.once('exit',()=>{clearTimeout(timeout);reject(Error('EARLY_RECEIPT_EXIT'));});
      child.once('message',m=>{
        clearTimeout(timeout);
        if(typeof m==='object' && m!==null && 'barrier' in m && m.barrier===input.cut) resolve();
        else reject(Error('RECEIPT_BARRIER'));
      });
      child.send(input);
    });
    assert.ok(child.kill('SIGKILL'));
  }finally{
    if(!child.killed) child.kill('SIGKILL');
    await exit;
  }
}
