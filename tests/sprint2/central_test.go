package sprint2
import("path/filepath";"testing";"zyposoft.local/operations/tests/fixtures";"zyposoft.local/operations/packages/local-policy";"zyposoft.local/operations/packages/signing-go")
func TestCentralReceiptBinding(t *testing.T){
 site:=fixture.NewSite();b:=site.Deployment(1,"product-test","LOCAL_BUSY","test-2","APP_CONFIG","g");path:=filepath.Join(t.TempDir(),"central.db")
 c,err:=OpenCentral(path,fixture.Scope,site.Config.Trust);if err!=nil{t.Fatal(err)}
 if err=c.Submit(b);err!=nil{t.Fatal(err)};if err=c.Submit(b);err!=nil{t.Fatal(err)}
 changed:=b;changed.Backup=nil;if err=c.Submit(changed);err==nil{t.Fatal("queue accepted conflicting bundle")}
 doc:=fixture.Doc(fixture.Example("MC-04"));body:=doc["body"].(signing.Object);body["job_id"]=fixture.Doc(b.Job)["body"].(signing.Object)["job_id"];artifact,_:=policy.Digest(b.Manifest);body["observed_sha256"]=artifact
 wire:=site.Sign(doc,"mock-receipt");if err=c.accept(wire);err!=nil{t.Fatal(err)};c.Close()
 c,err=OpenCentral(path,fixture.Scope,site.Config.Trust);if err!=nil{t.Fatal(err)};defer c.Close();if err=c.accept(wire);err!=nil{t.Fatal(err)}
 body["execution_id"]="77777777-7777-4777-8777-777777777777";if err=c.accept(site.Sign(doc,"mock-receipt"));err==nil{t.Fatal("conflicting signed outcome accepted")}
 body["observed_sha256"]="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";if err=c.accept(site.Sign(doc,"mock-receipt"));err==nil{t.Fatal("wrong artifact accepted")}
 var n int;if err=c.db.QueryRow("SELECT count(*) FROM terminal").Scan(&n);err!=nil||n!=1{t.Fatal(n,err)}
}
