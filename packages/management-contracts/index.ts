import {Ajv2020} from 'ajv/dist/2020.js';
import formats from 'ajv-formats';
import {readFileSync} from 'node:fs';
import catalog from './catalog.json' with {type:'json'};
import {parseWire, object} from './json.ts';
import type {Envelope, LocalPayloads} from './generated/types.ts';

const ajv = new Ajv2020({strict:true, strictRequired:false, strictTypes:false, ownProperties:true, addUsedSchema:false});
formats.default(ajv);
const schema = (path:string) => JSON.parse(readFileSync(new URL(path,import.meta.url),'utf8'));
export const contracts = new Map(catalog.map(entry => [entry.id,entry]));
const validators = new Map(catalog.map(entry => [entry.id, ajv.compile(schema(entry.schema))]));
const local = schema('schemas/LOCAL-01.product-management.schema.json');
ajv.addSchema(local);
export function decodeLocal<K extends keyof LocalPayloads>(endpoint: K, wire: Uint8Array|string): LocalPayloads[K] {
  const data = parseWire(wire);
  if (!Object.hasOwn(local.$defs, endpoint) || !ajv.validate(local.$id+'#/$defs/'+endpoint,data)) throw Error('INVALID_LOCAL_PAYLOAD');
  return data as LocalPayloads[K];
}
export function decode(wire: Uint8Array|string): Envelope {
  const data = object(parseWire(wire));
  if (data.schema_version !== '1.2.0') throw Error('UNKNOWN_VERSION');
  if (!validators.get(String(data.contract_id))?.(data)) throw Error('INVALID_CONTRACT');
  return data as Envelope;
}
