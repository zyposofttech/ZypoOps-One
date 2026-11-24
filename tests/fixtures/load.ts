import {readFileSync} from 'node:fs';
import {createPublicKey} from 'node:crypto';
import type {TrustedKey} from '../../packages/signing-ts/index.ts';
export const read = (file:string) => readFileSync(new URL(file,import.meta.url),'utf8');
export const example=(id:string)=>read('examples/'+id+'.example.json');
export const scope={tenant_id:'11111111-1111-4111-8111-111111111111',site_id:'22222222-2222-4222-8222-222222222222'};
export function fixtureTrust(): Map<string,TrustedKey> {
  const registry:Record<string,TrustedKey & {public_key_file:string}>=JSON.parse(read('test_key_registry.json'));
  return new Map(Object.entries(registry).map(([kid,key])=>[kid,{...key,...(key.purpose.startsWith('agent_')?scope:{}),publicKey:createPublicKey(read(key.public_key_file))}]));
}
