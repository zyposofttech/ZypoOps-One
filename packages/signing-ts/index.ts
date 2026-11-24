import {sign, verify, type KeyObject} from 'node:crypto';
import {contracts, decode} from '../management-contracts/index.ts';
import {canonical, object, parseWire, type JsonObject} from '../management-contracts/json.ts';

export interface Scope {tenant_id:string; site_id:string}
export interface TrustedKey {
  purpose:string; alg:'ES256'|'Ed25519'; publicKey:KeyObject;
  revoked?:boolean; tenant_id?:string; site_id?:string;
}
export type Trust = ReadonlyMap<string,TrustedKey>;
export interface Expected {contract:string; scope?:Scope}
type Signature = {kid:string; alg:'Ed25519'|'ES256'; value:string};

function without(value:JsonObject, field:string) {
  const copy={...value};
  delete copy[field];
  return copy;
}
function bytes(domain:string, data:JsonObject) {
  return Buffer.from(domain+'\0'+canonical(data),'utf8');
}
export function envelopeBytes(doc:JsonObject) {
  return bytes('ZOP-SIG-V1\0'+doc.contract_id,without(doc,'signature'));
}
function nested(doc:JsonObject) {
  const body=object(doc.body);
  if (doc.contract_id==='MC-09') {
    if(body.maker_id===body.checker_id) throw Error('MAKER_CHECKER_SAME');
    const field='site_authoriser_attestation';
    return {field, purpose:'site_authoriser', input:bytes('ZOP-SITE-CONSENT-V1',{
      tenant_id:doc.tenant_id,site_id:doc.site_id,body:without(body,field)})};
  }
  if (doc.contract_id==='MC-12') {
    const field='clinical_owner_signature';
    return {field,purpose:'clinical_owner',input:bytes('ZOP-CLINICAL-PACKAGE-V1',without(body,field))};
  }
}
function keyFor(trust:Trust,kid:string,alg:string,purpose:string,doc:JsonObject) {
  const key=trust.get(kid);
  if (!key || key.revoked || key.alg!==alg || key.purpose!==purpose) throw Error('UNTRUSTED_KEY');
  if ((purpose.startsWith('agent_') || purpose==='site_authoriser') && (!key.site_id || !key.tenant_id)) throw Error('UNSCOPED_KEY');
  if ((key.tenant_id!==undefined && key.tenant_id!==doc.tenant_id) || (key.site_id!==undefined && key.site_id!==doc.site_id)) throw Error('KEY_SCOPE');
  if (key.alg==='ES256' ? key.publicKey.asymmetricKeyDetails?.namedCurve!=='prime256v1' : key.publicKey.asymmetricKeyType!=='ed25519') throw Error('KEY_TYPE');
  return key;
}
function check(trust:Trust,doc:JsonObject,sig:Signature,purpose:string,input:Buffer) {
  const key=keyFor(trust,sig.kid,sig.alg,purpose,doc);
  const raw=Buffer.from(sig.value,'base64url');
  if(raw.length!==64 || raw.toString('base64url')!==sig.value) throw Error('SIGNATURE_FORMAT');
  if(!verify(sig.alg==='ES256'?'sha256':null,input,{key:key.publicKey,dsaEncoding:'ieee-p1363'},raw)) throw Error('BAD_SIGNATURE');
}
export function verifyMessage(wire:Uint8Array|string, expected:Expected, trust:Trust) {
  const doc=decode(wire);
  const entry=contracts.get(expected.contract);
  if(doc.contract_id!==expected.contract || !entry) throw Error('WRONG_CONTRACT');
  if(entry.scope==='site_scoped' && (!expected.scope || !('tenant_id' in doc) ||
    doc.tenant_id!==expected.scope.tenant_id || doc.site_id!==expected.scope.site_id)) throw Error('MESSAGE_SCOPE');
  check(trust,doc,doc.signature,entry.key_purpose,envelopeBytes(doc));
  const target=nested(doc);
  if(target) check(trust,doc,object(doc.body)[target.field] as Signature,target.purpose,target.input);
  return doc;
}
function signature(kid:string,key:TrustedKey,privateKey:KeyObject,data:Buffer):Signature {
  const value=sign(key.alg==='ES256'?'sha256':null,data,{key:privateKey,dsaEncoding:'ieee-p1363'}).toString('base64url');
  return {kid,alg:key.alg,value};
}
export function signMessage(unsigned:Uint8Array|string, expected:Expected, trust:Trust, kid:string, privateKey:KeyObject) {
  const doc=object(parseWire(unsigned));
  const key=trust.get(kid);
  if(!key || Object.hasOwn(doc,'signature') || doc.contract_id!==expected.contract) throw Error('SIGNING_INPUT');
  keyFor(trust,kid,key.alg,contracts.get(expected.contract)?.key_purpose??'',doc);
  doc.signature=signature(kid,key,privateKey,envelopeBytes(doc));
  const wire=canonical(doc);
  verifyMessage(wire,expected,trust);
  return wire;
}
export function signAttestation(unsigned:string, trust:Trust, kid:string, privateKey:KeyObject) {
  const doc=object(parseWire(unsigned)), target=nested(doc), key=trust.get(kid);
  if(!key || !target) throw Error('ATTESTATION_TYPE');
  keyFor(trust,kid,key.alg,target.purpose,doc);
  const sig=signature(kid,key,privateKey,target.input);
  check(trust,doc,sig,target.purpose,target.input);
  return sig;
}
