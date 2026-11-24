import assert from 'node:assert/strict';
import test from 'node:test';
import {createPublicKey,generateKeyPairSync,sign} from 'node:crypto';
import {decode, decodeLocal} from '../../packages/management-contracts/index.ts';
import {canonical,object,parseWire} from '../../packages/management-contracts/json.ts';
import {verifyMessage,envelopeBytes} from '../../packages/signing-ts/index.ts';
import {read,example,scope,fixtureTrust} from '../fixtures/load.ts';
const trust=fixtureTrust();
for(const v of JSON.parse(read('vectors/signed_vectors.json'))) test('golden '+v.contract_id,()=>{
  const wire=example(v.contract_id);
  verifyMessage(wire,{contract:v.contract_id,scope},trust);
  assert.equal(envelopeBytes(object(parseWire(wire))).toString('base64'),v.input_b64);
});
for(const n of JSON.parse(read('vectors/negative_signed_vectors.json'))) test(n.id,()=>{
  assert.throws(()=>verifyMessage(JSON.stringify(n.message),{contract:n.message.contract_id,scope:n.message},trust));
});
test('new central signature cannot move site consent',()=>{
  const {privateKey,publicKey}=generateKeyPairSync('ed25519');
  const doc=JSON.parse(example('MC-09'));
  doc.site_id='99999999-9999-4999-8999-999999999999';
  const keys=new Map(trust);
  keys.set(doc.signature.kid,{purpose:'ops_approval',alg:'Ed25519',publicKey});
  doc.signature.value=sign(null,envelopeBytes(doc),privateKey).toString('base64url');
  assert.throws(()=>verifyMessage(JSON.stringify(doc),{contract:'MC-09',scope:doc},keys),/KEY_SCOPE/);
  // Even a mis-bound copy of the public key cannot authenticate the old bytes.
  const kid=doc.body.site_authoriser_attestation.kid;
  keys.set(kid,{...keys.get(kid)!,site_id:doc.site_id});
  assert.throws(()=>verifyMessage(JSON.stringify(doc),{contract:'MC-09',scope:doc},keys),/BAD_SIGNATURE/);
});
test('JCS shared corpus and wire budgets',()=>{
  const corpus=JSON.parse(read('canonical.json'));
  for(const [wire,want] of corpus.valid) assert.equal(canonical(parseWire(wire)),want);
  for(const bad of [...corpus.invalid,'['.repeat(65)+']'.repeat(65),' '.repeat(1048577),Buffer.from([34,255,34])]) assert.throws(()=>parseWire(bad));
});
test('signature, trust and pinned wire version rejection',()=>{
  const doc=JSON.parse(example('MC-04')), verify=(keys=trust)=>verifyMessage(JSON.stringify(doc),{contract:'MC-04',scope},keys);
  for(const sig of [{...doc.signature,value:Buffer.alloc(70).toString('base64url')},{...doc.signature,alg:'Ed25519'},{...doc.signature,kid:'unknown'},undefined]){
    const old=doc.signature;doc.signature=sig;assert.throws(()=>verify());doc.signature=old;
  }
  const kid=doc.signature.kid, key=trust.get(kid)!;
  for(const change of [{revoked:true},{purpose:'ops_job'},{publicKey:createPublicKey(read('vectors/public_keys/test-only-ops-job.pem'))}]) {
    assert.throws(()=>verify(new Map(trust).set(kid,{...key,...change})));
  }
  doc.body.stage='ALTERED';assert.throws(()=>verify());doc.schema_version='1.3.0';assert.throws(()=>verify(),/UNKNOWN_VERSION/);
});
test('MC-10 nullable reference, length boundary and LOCAL-01 field rejection',()=>{
  const doc=JSON.parse(example('MC-10'));
  for(const v of [null,'x'.repeat(128)]) {doc.body.clinical_approval_ref=v;decode(JSON.stringify(doc))}
  doc.body.clinical_approval_ref='x'.repeat(129);assert.throws(()=>decode(JSON.stringify(doc)));
  const ready={ready:true,service_codes:['APP_READY'],version:'1.5'};
  decodeLocal('readiness_response',JSON.stringify(ready));
  assert.throws(()=>decodeLocal('readiness_response',JSON.stringify({...ready,patient_name:'synthetic'})));
});
for(const row of JSON.parse(read('command_params_vectors.json'))) test('command '+row.command_type,()=>{
  const doc=JSON.parse(example('MC-03'));
  Object.assign(doc.body,{command_type:row.command_type,params:row.valid_params});decode(JSON.stringify(doc));
  doc.body.params.extra=true;assert.throws(()=>decode(JSON.stringify(doc)));
});
