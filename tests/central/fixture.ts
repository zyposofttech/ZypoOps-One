import {createHmac, generateKeyPairSync, randomBytes, randomUUID, sign, type KeyObject} from 'node:crypto';
import {example} from '../fixtures/load.ts';
import {signAttestation, signMessage, type TrustedKey} from '../../packages/signing-ts/index.ts';
import {canon, hash, type Identity, type Role, type SyntheticConfig} from '../../services/ops-api/modules/context.ts';

export function fixture() {
  const trust=new Map<string,TrustedKey>(), priv=new Map<string,KeyObject>();
  function key(purpose:string,site?:{tenantId:string;siteId:string}) {
    const kid=purpose+'-'+randomUUID(),pair=generateKeyPairSync('ed25519');
    trust.set(kid,{purpose,alg:'Ed25519',publicKey:pair.publicKey,...(site?{tenant_id:site.tenantId,site_id:site.siteId}:{})});priv.set(kid,pair.privateKey);return kid;
  }
  const global={identity:key('ops_identity'),job:key('ops_job'),release:key('release_signer'),entitlement:key('entitlement_signer'),approval:key('ops_approval')};
  const identities:{tokenHash:string;identity:Identity}[]=[];
  const ta=randomUUID(),tb=randomUUID(),sa=randomUUID();
  const installs=[{tenantId:ta,siteId:sa},{tenantId:ta,siteId:sa},{tenantId:ta,siteId:randomUUID()},{tenantId:tb,siteId:randomUUID()}].map((site,i)=>{
    const install=randomUUID(),agent=randomUUID();
    function principal(role:Role) {
      const token=randomBytes(32).toString('hex'),identity={...site,role,actor:randomUUID(),...(role==='agent'?{installation:install,agent}:{})};
      identities.push({tokenHash:hash(token),identity});return {token,identity};
    }
    const keys={agent,pop:key('agent_enrolment_pop',site),receipt:key('agent_receipt',site),backup:key('agent_backup_receipt',site),progress:key('agent_progress',site),telemetry:key('agent_telemetry',site)};
    const secondary=i%2===1;
    return {...site,install,agent,keys,product:secondary?'mock-secondary':'zyposono-sim',release:secondary?'worker-test-5':'sono-test-2',lock:secondary?'BATCH_IN_PROGRESS':'ACTIVE_ACQUISITION',category:secondary?'MANAGEMENT_AGENT':'APP_CONFIG',authoriser:randomUUID(),consent:key('site_authoriser',site),maker:principal('operator'),checker:principal('operator'),provisioner:principal('provisioner'),installer:principal('installer'),device:principal('agent')};
  });
  const config:SyntheticConfig={trust,identities,targets:new Map(installs.map(i=>[i.install,i.keys])),claimKey:randomBytes(32),csrVerifier:process.env.SYNTHETIC_CSR!,provisionUrl:process.env.PROVISION_URL!,opsIdentity:{kid:global.identity,privateKey:priv.get(global.identity)!},opsJob:{kid:global.job,privateKey:priv.get(global.job)!}};
  function unsigned(contract:string,site:typeof installs[number],body:Record<string,unknown>) {
    const doc=JSON.parse(example(contract));
    delete doc.signature;doc.sent_at=new Date().toISOString();doc.message_id=randomUUID();
    if(contract!=='MC-05'){doc.tenant_id=site.tenantId;doc.site_id=site.siteId;}
    Object.assign(doc.body,body);return doc;
  }
  function signed(doc:ReturnType<typeof unsigned>,kid:string) {
    return signMessage(canon(doc),{contract:doc.contract_id,scope:{tenant_id:doc.tenant_id,site_id:doc.site_id}},trust,kid,priv.get(kid)!);
  }
  function grant(site:typeof installs[number],request:string,digest:string,extra={}) {
    const m=unsigned('MC-09',site,{grant_id:request,site_authoriser_id:site.authoriser,resource_scope:[site.product],operator_id:site.maker.identity.actor,maker_id:site.maker.identity.actor,checker_id:site.checker.identity.actor,target_digest_sha256:digest,valid_from:new Date(Date.now()-10000).toISOString(),valid_until:new Date(Date.now()+600000).toISOString(),...extra});
    delete m.body.site_authoriser_attestation;
    m.body.site_authoriser_attestation=signAttestation(canon(m),trust,site.consent,priv.get(site.consent)!);
    return signed(m,global.approval);
  }
  function enrollment(site:typeof installs[number],claim:{id:string;nonce:string;bootstrapSecret:string},extra={}) {
    // A minimal genuine Ed25519 PKCS#10 test request with an empty subject and attributes.
    const pair=trust.get(site.keys.pop)!.publicKey.export({type:'spki',format:'der'});
    const node=(tag:number,body:Buffer)=>Buffer.concat([Buffer.from(body.length<128?[tag,body.length]:[tag,0x81,body.length]),body]);
    const cri=node(0x30,Buffer.concat([Buffer.from('0201003000','hex'),pair,Buffer.from('a000','hex')]));
    const signature=sign(null,cri,priv.get(site.keys.pop)!);
    const der=node(0x30,Buffer.concat([cri,Buffer.from('300506032b6570','hex'),node(3,Buffer.concat([Buffer.from([0]),signature]))]));
    const csr=der.toString('base64').match(/.{1,64}/g)!.join('\n');
    const binding={claim_id:claim.id,installation_id:site.install,claim_nonce:claim.nonce,csr_sha256:hash(der),site_id:site.siteId};
    const mac=createHmac('sha256',Buffer.from(claim.bootstrapSecret,'hex')).update(canon(binding)).digest('base64url');
    return signed(unsigned('MC-02',site,{claim_id:claim.id,installation_id:site.install,installer_id:site.installer.identity.actor,claim_nonce:claim.nonce,csr_pem:'-----BEGIN CERTIFICATE REQUEST-----\n'+csr+'\n-----END CERTIFICATE REQUEST-----\n',csr_sha256:binding.csr_sha256,token_proof:mac,agent_version:'1.2.0',...extra}),site.keys.pop);
  }
  return {config,trust,priv,global,installs,unsigned,signed,grant,enrollment};
}
