import {PrismaClient, type Prisma} from '@prisma/client';
export interface SiteContext {tenantId:string;siteId:string}
const uuid=/^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i;
// Context is supplied by the authenticated caller. Only this transaction client reaches the service.
export class TenantDatabase {
  #client:PrismaClient;
  constructor(client=new PrismaClient()){this.#client=client}
  onModuleDestroy(){return this.close()}
  close(){return this.#client.$disconnect()}
  run<T>(scope:SiteContext,operation:(tx:Prisma.TransactionClient)=>Promise<T>):Promise<T>{
    if (!scope || !uuid.test(scope.tenantId) || !uuid.test(scope.siteId)) throw Error('INVALID_SITE_CONTEXT');
    return this.#client.$transaction(async tx=>{
      // set_config(..., true) is parameterized SET LOCAL, including on pooled connections.
      await tx.$queryRaw`SELECT set_config('app.tenant_id',${scope.tenantId},true),set_config('app.site_id',${scope.siteId},true) AS site_context`;
      return operation(tx);
    },{maxWait:30000,timeout:10000});
  }
}
