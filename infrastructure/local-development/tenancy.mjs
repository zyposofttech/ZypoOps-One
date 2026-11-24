// Creates a fresh loopback-only, SCRAM-authenticated PG16 cluster. Never accepts an existing database URL.
import {execFileSync} from 'node:child_process';
import {mkdtempSync,mkdirSync,writeFileSync,rmSync} from 'node:fs';
import {randomBytes} from 'node:crypto';
import {createServer} from 'node:net';
import path from 'node:path';
import assert from 'node:assert/strict';
const bin=name=>path.join(process.env.PGBIN||'',name);
assert.match(execFileSync(bin('postgres'),['--version'],{encoding:'utf8'}),/PostgreSQL\) 16\./);
mkdirSync('.work',{recursive:true});
const dir=path.resolve(mkdtempSync('.work/pg16-')),data=path.join(dir,'data');
const passwords=Array.from({length:3},()=>randomBytes(32).toString('hex'));
const bind=createServer();await new Promise(resolve=>bind.listen(0,'127.0.0.1',resolve));
const port=bind.address().port;await new Promise(resolve=>bind.close(resolve));
const pgpass=path.join(dir,'init-password');
writeFileSync(pgpass,passwords[0],{mode:0o600});
const dbUrl=(user,password)=>'postgresql://'+user+':'+password+'@127.0.0.1:'+port+'/postgres?schema=zop&connection_limit=4';
const sql=(user,password,input)=>execFileSync(bin('psql'),['-X','-q','-v','ON_ERROR_STOP=1','-h','127.0.0.1','-p',String(port),'-U',user,'-d','postgres'],{input,env:{...process.env,PGPASSWORD:password}});
const prisma=(args,env)=>execFileSync(process.execPath,['node_modules/prisma/build/index.js',...args,'--schema','packages/database/schema.prisma'],{env,stdio:'inherit'});
let started=false;
try{
 execFileSync(bin('initdb'),['-D',data,'-U','test_admin','--auth=scram-sha-256','--encoding=UTF8','--locale=C','--pwfile='+pgpass],{stdio:'inherit'});
 rmSync(pgpass);
 execFileSync(bin('pg_ctl'),['-D',data,'-l',path.join(dir,'server.log'),'-o','-h 127.0.0.1 -p '+port,'-w','start'],{stdio:'inherit'});started=true;
 sql('test_admin',passwords[0],"CREATE ROLE zop_migrate LOGIN NOSUPERUSER NOCREATEROLE NOBYPASSRLS PASSWORD '"+passwords[1]+"'; CREATE ROLE zop_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS PASSWORD '"+passwords[2]+"'; GRANT CREATE ON DATABASE postgres TO zop_migrate; CREATE SCHEMA zop AUTHORIZATION zop_migrate;");
 const env={...process.env,DATABASE_URL:dbUrl('zop_migrate',passwords[1])};
 prisma(['migrate','deploy'],env);prisma(['migrate','deploy'],env);
 if(process.argv.includes('--introspect')) prisma(['db','pull'],env);
 prisma(['generate'],env);
 execFileSync(process.execPath,['--test','tests/tenancy/isolation.test.ts'],{env:{...env,TEST_ADMIN_URL:dbUrl('test_admin',passwords[0]),DATABASE_URL:dbUrl('zop_runtime',passwords[2])},stdio:'inherit'});
} finally {
 if(started)execFileSync(bin('pg_ctl'),['-D',data,'-m','fast','-w','stop'],{stdio:'inherit'});
 rmSync(dir,{recursive:true,force:true});
}
