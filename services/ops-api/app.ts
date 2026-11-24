import 'reflect-metadata';
import {Module} from '@nestjs/common';
import {NestFactory} from '@nestjs/core';
import {TenantDatabase} from '../../packages/database/tenant.ts';
// Internal application context only. Authenticated ingress depends on OD05.
export class OperationsModule {}
Module({providers:[{provide:TenantDatabase,useFactory:()=>new TenantDatabase()}],exports:[TenantDatabase]})(OperationsModule);
export const createOperationsContext=()=>NestFactory.createApplicationContext(OperationsModule,{logger:false});
