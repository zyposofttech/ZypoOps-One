//go:build linux
package sprint2
import("context";"path/filepath";"testing";"time";"strings";"sync";"crypto/rand";"zyposoft.local/operations/packages/deployment";pl "zyposoft.local/operations/packages/product-local";"zyposoft.local/operations/packages/signing-go";"zyposoft.local/operations/services/site-agent";"zyposoft.local/operations/tests/fixtures")
func TestAdmissionFailures(t *testing.T){
 cases:=[]string{"tampered job","expired job","wrong tenant","wrong site","wrong key purpose","unapproved artifact","cross-product binding","missing backup","old backup","wrong backup purpose","tampered backup","wrong dataset","unencrypted","failed integrity","backup from another operation","missing consent","active lock","product unready","malformed locks","local timeout"}
 for _,name:=range cases{t.Run(name,func(t *testing.T){
  dir:=testDir(t);p:=makePlan(dir,true);apps:=startApps(t,dir,p);b:=p.jobs[0];site:=p.site
  job:=fixture.Doc(b.Job);j:=job["body"].(signing.Object);backup:=fixture.Doc(b.Backup);e:=backup["body"].(signing.Object)
  switch name{
  case "tampered job":j["sequence"]=33;b.Job=fixture.JSON(job)
  case "expired job":j["expires_at"]="2026-11-24T05:00:10Z";b.Job=site.Sign(job,"mock-job")
  case "wrong tenant","wrong site":field:="site_id";if name=="wrong tenant"{field="tenant_id"};job[field]="99999999-9999-4999-8999-999999999999";b.Job=site.Sign(job,"mock-job")
  case "wrong key purpose":signer:=fixture.NewSite();key:=signer.Config.Trust["mock-grant"];key.Purpose="ops_job";signer.Config.Trust["mock-grant"]=key;b.Job=signer.Sign(job,"mock-grant")
  case "unapproved artifact":delete(site.Config.TestArtifacts,p.profiles[0].Artifact)
  case "cross-product binding":j["params"].(signing.Object)["product_key"]=p.profiles[1].Product;b.Job=site.Sign(job,"mock-job")
  case "missing backup":b.Backup=nil
  case "old backup":e["last_success_at"]="2026-11-21T05:00:00Z";b.Backup=site.Sign(backup,"mock-backup")
  case "wrong backup purpose":signer:=fixture.NewSite();key:=signer.Config.Trust["mock-receipt"];key.Purpose="agent_backup_receipt";signer.Config.Trust["mock-receipt"]=key;b.Backup=signer.Sign(backup,"mock-receipt")
  case "tampered backup":e["backup_age_seconds"]=10;b.Backup=fixture.JSON(backup)
  case "wrong dataset":e["dataset_categories"]=[]string{"FINANCIAL_DB"};b.Backup=site.Sign(backup,"mock-backup")
  case "unencrypted":e["encrypted"]=false;b.Backup=site.Sign(backup,"mock-backup")
  case "failed integrity":e["integrity_verified"]=false;b.Backup=site.Sign(backup,"mock-backup")
  case "backup from another operation":b.Backup=p.jobs[1].Backup
  case "missing consent":b.Grants=nil
  case "active lock","product unready","malformed locks","local timeout":
   mode:=map[string]string{"malformed locks":"MALFORMED","local timeout":"TIMEOUT"}[name];if err:=apps[0].SetActivity(name=="active lock",name!="product unready",mode);err!=nil{t.Fatal(err)}
  }
  socket:=serveHelper(t,dir,p)
  a,err:=agent.OpenConnected(filepath.Join(dir,"agent.db"),ident(),site.Config,site,nil,p.apps,socket);if err!=nil{t.Fatal(err)};defer a.Close()
  if _,err=a.Handle(context.Background(),b,clock());err==nil{t.Fatal("agent accepted invalid request")}
  reply:=helperReply(t,socket,"execute",b);if reply.Status!="REFUSED"&&reply.Status!="DENIED"{t.Fatal("helper accepted invalid request",reply.Status)}
  for _,profile:=range p.profiles{if n:=scalar(t,filepath.Join(dir,profile.Product+".db"),"SELECT count(*) FROM effect");n!=0{t.Fatal(profile.Product,n)}}
  if n:=scalar(t,filepath.Join(dir,"helper.db"),"SELECT count(*) FROM consumed_grant");n!=0{t.Fatal(n)}
 })}
}
type beforeApply struct{pl.Application;event func()}
func(c beforeApply)Apply(ctx context.Context,a pl.Activation)(pl.Observation,error){c.event();return c.Application.Apply(ctx,a)}
func TestEffectBoundaries(t *testing.T){
 for _,mode:=range []string{"LOCK_AFTER_PREFLIGHT","AFTER_EFFECT","POST_UNREADY","POST_VERIFY_OFFLINE"}{t.Run(mode,func(t *testing.T){
 dir:=testDir(t);p:=makePlan(dir,true);apps:=startApps(t,dir,p);profile:=p.profiles[0];agentProducts:=makePlan(dir,false).apps
 if mode=="LOCK_AFTER_PREFLIGHT"{p.apps[profile.Product]=beforeApply{p.apps[profile.Product],func(){if err:=apps[0].SetActivity(true,true,"");err!=nil{t.Error(err)}}}}else if err:=apps[0].SetActivity(false,true,mode);err!=nil{t.Fatal(err)}
 socket:=serveHelper(t,dir,p);a,err:=agent.OpenConnected(filepath.Join(dir,"agent.db"),ident(),p.site.Config,p.site,nil,agentProducts,socket);if err!=nil{t.Fatal(err)};defer a.Close()
 wire,err:=a.Handle(context.Background(),p.jobs[0],clock());if err!=nil{t.Fatal(err)};doc,err:=signing.Verify("MC-04",wire,fixture.Scope,p.site.Config.Trust);if err!=nil{t.Fatal(err)};r:=doc["body"].(signing.Object)
 wantState,wantStage,effects:="FAILED","POST_VERIFY",1
 switch mode{case "LOCK_AFTER_PREFLIGHT":wantState,wantStage,effects="REJECTED","BEFORE_EFFECT",0;case "AFTER_EFFECT":wantStage="AFTER_EFFECT";case "POST_VERIFY_OFFLINE":wantState="UNKNOWN"}
 if r["state"]!=wantState||r["stage"]!=wantStage||r["observed_sha256"]!=profile.Artifact{t.Fatal(r)}
 replay:=helperReply(t,socket,"execute",p.jobs[0]);if replay.Result==nil||replay.Result.Execution!=r["execution_id"]||replay.Result.State!=wantState{t.Fatal(replay)}
 if n:=scalar(t,filepath.Join(dir,profile.Product+".db"),"SELECT count(*) FROM effect");n!=effects{t.Fatal(n)}
 })}
}
func TestConcurrentDuplicates(t *testing.T){
 dir:=testDir(t);p:=makePlan(dir,true);startApps(t,dir,p)
 // A new, signed job with its own backup attempts to reuse the already consumed grant.
 repeat:=p.site.Deployment(3,p.profiles[0].Product,p.profiles[0].Lock,p.profiles[0].Release,"APP_CONFIG",p.profiles[0].Product)
 socket:=serveHelper(t,dir,p);var wg sync.WaitGroup;ids:=make(chan string,12)
 for range 12{wg.Add(1);go func(){defer wg.Done();r,err:=socket.Exchange(context.Background(),deployment.Request{Operation:"execute",Bundle:p.jobs[0]});if err!=nil||r.Result==nil{t.Error("duplicate",r.Status,err);return};ids<-r.Result.Execution}()};wg.Wait();close(ids)
 var first string;for id:=range ids{if first==""{first=id};if id!=first{t.Fatal("new execution")}}
 if first==""||helperReply(t,socket,"execute",repeat).Status!="REFUSED"{t.Fatal("grant reuse accepted")}
 if n:=scalar(t,filepath.Join(dir,"helper.db"),"SELECT count(*) FROM consumed_grant");n!=1{t.Fatal(n)}
 if n:=scalar(t,filepath.Join(dir,p.profiles[0].Product+".db"),"SELECT count(*) FROM effect");n!=1{t.Fatal(n)}
}
func TestDisconnectedBeforeExecution(t *testing.T){
 dir:=testDir(t);p:=makePlan(dir,true);apps:=startApps(t,dir,p)
 socket:=deployment.Socket{Endpoint:endpoint(filepath.Join(dir,"h.sock"))}
 a,err:=agent.OpenConnected(filepath.Join(dir,"agent.db"),ident(),p.site.Config,p.site,nil,p.apps,socket);if err!=nil{t.Fatal(err)};defer a.Close()
 if _,err=a.Handle(context.Background(),p.jobs[0],clock());err==nil{t.Fatal("socket was absent")}
 ctx,cancel:=context.WithCancel(context.Background());cancel()
 if _,err=socket.Exchange(ctx,deployment.Request{Operation:"lookup",Bundle:p.jobs[0]});err==nil{t.Fatal("cancellation ignored")}
 if n:=scalar(t,filepath.Join(dir,p.profiles[0].Product+".db"),"SELECT count(*) FROM effect");n!=0{t.Fatal(n)}
 if err=apps[0].SetActivity(true,true,"");err!=nil{t.Fatal(err)}
 serveHelper(t,dir,p);wire,err:=a.Handle(context.Background(),p.jobs[0],clock());if err!=nil{t.Fatal(err)}
 if fixture.Doc(wire)["body"].(signing.Object)["state"]!="REJECTED"{t.Fatal("unfenced cancellation")}
 if err=apps[0].SetActivity(false,true,"");err!=nil{t.Fatal(err)}
 if helperReply(t,socket,"execute",p.jobs[0]).Status!="REFUSED"{t.Fatal("late duplicate executed")}
}
func TestLocalAuthority(t *testing.T){
 dir:=testDir(t);p:=makePlan(dir,true);startApps(t,dir,p);profile:=p.profiles[0];client:=p.apps[profile.Product].(pl.Client)
 keyed:=client;keyed.WriteToken=rand.Text();a:=pl.Activation{Product:profile.Product,Artifact:profile.Artifact,Release:profile.Release,Job:"77777777-7777-4777-8777-777777777777",Execution:"66666666-6666-4666-8666-666666666666",Expires:fixture.Now.Add(time.Hour).Format(time.RFC3339)}
 if _,err:=keyed.Apply(context.Background(),a);err==nil{t.Fatal("reader mutated application")}
 a.Product=p.profiles[1].Product;if _,err:=client.Apply(context.Background(),a);err==nil{t.Fatal("cross-product mutation")}
 client.Endpoint.Peer.GID++;if _,err:=client.ReadyVersion(context.Background(),"");err==nil{t.Fatal("wrong server credentials")}
 central,err:=OpenCentral(filepath.Join(dir,"central.db"),fixture.Scope,p.site.Config.Trust);if err!=nil{t.Fatal(err)};central.Close()
 other:=fixture.Scope;other.Tenant=strings.Repeat("b",36);if db,err:=OpenCentral(filepath.Join(dir,"central.db"),other,p.site.Config.Trust);err==nil{db.Close();t.Fatal("foreign store reopened")}
}

