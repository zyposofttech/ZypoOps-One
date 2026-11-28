import {Module} from '@nestjs/common';
import {randomUUID} from 'node:crypto';
import {NestFactory} from '@nestjs/core';
import {OperationsModule} from '../app.ts';
import {TenantDatabase} from '../../../packages/database/tenant.ts';
import {SyntheticContext, ServiceError, audit, type Identity, type SyntheticConfig} from './context.ts';
import {RegistryService} from './registry.ts';
import {EnrolmentService} from './enrolment.ts';

import {CatalogueService} from './catalogue.ts';
import {JobService} from './jobs.ts';
import {EvidenceService} from './evidence.ts';
import {ReadinessService} from './readiness.ts';

import {OnboardingService} from './onboarding.ts';

// Bound internal methods keep input strings and database errors out of denial events.
function secured<T extends object>(service:T,ctx:SyntheticContext):T {
  return new Proxy(service,{get(target,key,receiver){
    const fn=Reflect.get(target,key,receiver);
    if(typeof fn!=='function'||key==='constructor') return fn;
    return async (...args:unknown[])=>{
      try { return await Reflect.apply(fn,target,args); } catch(error:unknown) {
        const p=args[0] as Identity;
        ctx.authorise(p,['operator','agent','installer','provisioner']);
        const reason=error instanceof Error && /^[A-Z][A-Z0-9_]{1,60}$/.test(error.message)?error.message:'OPERATION_REJECTED';
        const correlation=randomUUID();
        // A site must exist for its append-only scoped audit history.
        await ctx.database.run(p,async tx=>{
          if(await tx.site.count({where:{id:p.siteId}})) await audit(tx,p,'DENIED_'+String(key).toUpperCase(),p.actor,correlation,'DENIED',reason);
        });
        throw new ServiceError(reason);
      }
    };
  }});
}

export async function createSyntheticApplication(config:SyntheticConfig) {
  class SyntheticModule {}
  const services=[RegistryService,EnrolmentService,CatalogueService,JobService,EvidenceService,ReadinessService,OnboardingService];
  Module({imports:[OperationsModule],providers:[
    {provide:SyntheticContext,inject:[TenantDatabase],useFactory:(db:TenantDatabase)=>new SyntheticContext(db,config)},
    ...services.map(service=>({provide:service,inject:[SyntheticContext],useFactory:(ctx:SyntheticContext)=>secured(new service(ctx),ctx)}))
  ]})(SyntheticModule);
  return NestFactory.createApplicationContext(SyntheticModule,{logger:false,abortOnError:false});
}
