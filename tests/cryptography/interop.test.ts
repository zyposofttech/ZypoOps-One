import {execFileSync} from 'node:child_process';
import {randomUUID,generateKeyPairSync,createPublicKey} from 'node:crypto';
import test from 'node:test';
import assert from 'node:assert/strict';
import {example,scope} from '../fixtures/load.ts';
import {signMessage,verifyMessage} from '../../packages/signing-ts/index.ts';
for (const alg of ['ES256','Ed25519'] as const) test('TS to Go to TS '+alg,()=>{
  const pair=alg==='ES256'?generateKeyPairSync('ec',{namedCurve:'P-256'}):generateKeyPairSync('ed25519');
  const record={...scope,alg,purpose:'agent_receipt',publicKey:pair.publicKey};
  const keys=new Map([['ts-test',record]]);
  const doc=JSON.parse(example('MC-04'));delete doc.signature;doc.message_id=randomUUID();
  const expected={contract:'MC-04',scope};
  const message=JSON.parse(signMessage(JSON.stringify(doc),expected,keys,'ts-test',pair.privateKey));
  const input=JSON.stringify({alg,message,public:pair.publicKey.export({type:'spki',format:'der'}).toString('base64')});
  const answer=JSON.parse(execFileSync('go',['run','./tests/cryptography/bridge'],{input,encoding:'utf8'}));
  keys.set('go-test',{...record,publicKey:createPublicKey({key:Buffer.from(answer.public,'base64'),type:'spki',format:'der'})});
  const verified=verifyMessage(JSON.stringify(answer.message),expected,keys);
  assert.equal(verified.message_id,doc.message_id);assert.equal(verified.signature.kid,'go-test');
});
