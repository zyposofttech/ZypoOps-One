export type Json = null | boolean | number | string | Json[] | JsonObject;
export type JsonObject = { [key: string]: Json };
export const maxBytes = 1024 * 1024;
export const maxDepth = 64;

export function object(value: Json | undefined): JsonObject {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) throw Error('EXPECTED_OBJECT');
  return value;
}

// Parse the wire representation before JSON.parse can discard duplicate member names.
export function parseWire(input: string | Uint8Array): Json {
  if ((typeof input === 'string' ? Buffer.byteLength(input) : input.length) > maxBytes) throw Error('PAYLOAD_LIMIT');
  const text = typeof input === 'string' ? input : new TextDecoder('utf-8', {fatal:true, ignoreBOM:true}).decode(input);
  // RFC 8259 excludes literal control characters from JSON strings.
  // eslint-disable-next-line no-control-regex
  const token = /[ \t\r\n]*(?:(?<str>"(?:[^"\\\x00-\x1f]|\\(?:["\\/bfnrt]|u[0-9a-fA-F]{4}))*")|(?<primitive>-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null)|(?<symbol>[[\]{}:,]))/y;
  let offset = 0;
  function next(): string {
    token.lastIndex = offset;
    const match = token.exec(text);
    if (!match) throw Error('INVALID_JSON');
    offset = token.lastIndex;
    return match[0].trimStart();
  }
  function string(value: string): string {
    if (!value.startsWith('"')) throw Error('INVALID_JSON');
    const decoded: string = JSON.parse(value);
    if (!decoded.isWellFormed()) throw Error('INVALID_UNICODE');
    return decoded;
  }
  function read(value: string, depth: number): Json {
    if (depth > maxDepth) throw Error('NESTING_LIMIT');
    if (value.startsWith('"')) return string(value);
    if (value === '{' || value === '[') {
      if (depth >= maxDepth) throw Error('NESTING_LIMIT');
      const array = value === '[';
      const result: JsonObject = Object.create(null);
      const items: Json[] = [];
      let item = next();
      const end = array ? ']' : '}';
      if (item !== end) for (;;) {
        if (array) items.push(read(item, depth + 1));
        else {
          const key = string(item);
          if (Object.hasOwn(result,key)) throw Error('DUPLICATE_KEY');
          if (next() !== ':') throw Error('INVALID_JSON');
          result[key] = read(next(), depth + 1);
        }
        item = next();
        if (item === end) break;
        if (item !== ',') throw Error('INVALID_JSON');
        item = next();
      }
      return array ? items : result;
    }
    if ('}]:,'.includes(value)) throw Error('INVALID_JSON');
    const primitive = JSON.parse(value);
    if (typeof primitive === 'number' && !Number.isFinite(primitive)) throw Error('INVALID_NUMBER');
    return primitive;
  }
  const value = read(next(), 0);
  if (!/^[ \r\n\t]*$/.test(text.slice(offset))) throw Error('TRAILING_JSON');
  return value;
}

export function canonical(value: Json, depth = 0): string {
  if (depth > maxDepth) throw Error('NESTING_LIMIT');
  if (Array.isArray(value)) return '[' + value.map(v => canonical(v, depth + 1)).join(',') + ']';
  if (value !== null && typeof value === 'object') {
    return '{' + Object.keys(value).sort().map(k => canonical(k) + ':' + canonical(value[k],depth+1)).join(',') + '}';
  }
  if (typeof value === 'string' && !value.isWellFormed()) throw Error('INVALID_UNICODE');
  if (typeof value === 'number' && !Number.isFinite(value)) throw Error('INVALID_NUMBER');
  return JSON.stringify(value);
}
