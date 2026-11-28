import {createHash, randomUUID, type KeyObject} from 'node:crypto';
import {PrismaClient, type Prisma} from '@prisma/client';
import {TenantDatabase, type SiteContext} from '../../../packages/database/tenant.ts';
import {canonical, type Json} from '../../../packages/management-contracts/json.ts';
import type {Trust} from '../../../packages/signing-ts/index.ts';

export class ServiceError extends Error {
  readonly code: string;
  constructor(code: string) { super(code); this.code = code; }
}
export function requireThat(ok: unknown, code: string): asserts ok {
  if (!ok) throw new ServiceError(code);
}
export const hash = (value: string | Buffer) => createHash('sha256').update(value).digest('hex');
export const canon = (value: unknown) => canonical(value as Json);
export function uuid(value: string) {
  requireThat(typeof value === 'string' && /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(value), 'INVALID_UUID');
  return value;
}
export function ref(value: string) {
  requireThat(typeof value === 'string' && /^[A-Za-z0-9][A-Za-z0-9_./:-]{0,119}$/.test(value), 'INVALID_REFERENCE');
  return value;
}
export function exact(data: object, fields: string[]) {
  requireThat(data && Object.keys(data).length === fields.length && fields.every(f => Object.hasOwn(data,f)), 'INVALID_INPUT');
}
export type Tx = Prisma.TransactionClient;
export type Role = 'provisioner' | 'operator' | 'installer' | 'agent';
export interface Identity extends SiteContext {actor: string; role: Role; installation?: string; agent?: string}
export interface TestSigner {kid: string; privateKey: KeyObject}
export interface SyntheticConfig {
  identities: ReadonlyArray<{tokenHash: string; identity: Identity}>;
  trust: Trust;
  targets: ReadonlyMap<string,{agent:string;pop:string;receipt:string;telemetry:string;backup:string;progress:string}>;
  opsIdentity: TestSigner;
  opsJob: TestSigner;
  claimKey: Buffer;
  provisionUrl: string;
  csrVerifier: string;
}

export class SyntheticContext {
  readonly provision: TenantDatabase;
  readonly config: SyntheticConfig;
  readonly database: TenantDatabase;
  #tokens = new Map<string,Readonly<Identity>>();
  #issued = new WeakSet<object>();
  constructor(database: TenantDatabase, config: SyntheticConfig) {
    requireThat(process.env.NODE_ENV === 'test' && config.claimKey.length === 32 && config.identities.length > 0, 'TEST_MODE_REQUIRED');
    const runtime = new URL(process.env.DATABASE_URL??'about:blank');
    requireThat(runtime.username==='zop_runtime' && ['127.0.0.1','[::1]'].includes(runtime.hostname),'RUNTIME_DATABASE');
    const url = new URL(config.provisionUrl);
    requireThat(url.username === 'zop_synthetic_provision' && ['127.0.0.1','[::1]'].includes(url.hostname), 'PROVISION_DATABASE');
    this.config = config;
    this.database = database;
    this.provision = new TenantDatabase(new PrismaClient({datasourceUrl:config.provisionUrl}));
    for (const {tokenHash,identity} of config.identities) {
      requireThat(/^[a-f0-9]{64}$/.test(tokenHash) && !this.#tokens.has(tokenHash),'IDENTITY_CONFIG');
      uuid(identity.tenantId); uuid(identity.siteId); uuid(identity.actor);
      requireThat(['provisioner','operator','installer','agent'].includes(identity.role),'IDENTITY_CONFIG');
      if(identity.role === 'agent') { uuid(identity.installation!); uuid(identity.agent!); }
      const principal = Object.freeze({...identity});
      this.#tokens.set(tokenHash, principal);
      this.#issued.add(principal);
    }
  }
  authenticate(token: string): Readonly<Identity> {
    requireThat(typeof token === 'string' && /^[a-f0-9]{64}$/.test(token),'UNAUTHENTICATED');
    const principal = this.#tokens.get(hash(token));
    requireThat(principal,'UNAUTHENTICATED');
    return principal;
  }
  authorise(p: Identity, roles: Role[]) {
    requireThat(p && this.#issued.has(p) && roles.includes(p.role),'FORBIDDEN');
  }
  onModuleDestroy() { return this.provision.close(); }
}
export const scope = (p: Identity) => ({tenant_id:p.tenantId,site_id:p.siteId});
export async function audit(tx:Tx,p:Identity,action:string,resource:string,correlation:string,outcome='OK',reason='NONE') {
  // A transaction-level advisory lock serializes the per-site chain under pooled writers.
  await tx.$executeRaw`SELECT pg_advisory_xact_lock(hashtextextended(${p.siteId},1))`;
  const prior = await tx.operational_audit_event.findFirst({where:scope(p),orderBy:{event_seq:'desc'}});
  const previous = prior?.event_sha256 ?? '0'.repeat(64);
  const seq = (prior?.event_seq ?? 0n) + 1n;
  const now = new Date();
  const entry = {id:randomUUID(),...scope(p),event_code:action,actor_subject:p.actor,resource_id:resource,correlation_id:correlation,outcome,reason_code:reason,created_at:now};
  const digest = hash(canon({...entry,created_at:now.toISOString(),event_seq:seq.toString(),previous_chain_sha256:previous}));
  await tx.operational_audit_event.create({data:{...entry,event_seq:seq,previous_chain_sha256:previous,event_sha256:digest}});
}
export async function activeAgent(tx:Tx,p:Identity) {
  // Hold identity state stable until this authenticated ingestion commits.
  await tx.$queryRaw`SELECT id FROM zop.site_agent WHERE id=${p.agent}::uuid AND installation_id=${p.installation}::uuid FOR SHARE`;
  const agent=await tx.site_agent.findFirst({where:{...scope(p),id:p.agent,installation_id:p.installation,status:'ENROLLED'}});
  requireThat(p.role==='agent' && agent,'IDENTITY_UNAVAILABLE');
  return agent;
}
