package helper
import("path/filepath";"testing";"time";"zyposoft.local/operations/tests/fixtures";"zyposoft.local/operations/packages/signing-go";"zyposoft.local/operations/packages/local-policy")
func TestContainment(t *testing.T){
 s:=fixture.NewSite();now:=fixture.Now;path:=filepath.Join(t.TempDir(),"helper.db");cfg:=s.Config.Clone()
 h,err:=Open(path,cfg,1001,1002,func()time.Time{return now});if err!=nil{t.Fatal(err)}
 good:=s.Job(1,true,[]string{"grant-a","grant-b"},"mock-job")
 raw:=fixture.JSON(good);for _,peer:=range [][2]uint32{{1,1002},{1001,1}}{if code:=h.apply(raw,peer[0],peer[1]);code!="UNAUTHORISED_PEER"{t.Fatal(code)}}
 for _,k:=range []string{"mount","executable"}{req:=fixture.Doc(raw);req[k]="/test/forbidden";if h.apply(fixture.JSON(req),1001,1002)!="INVALID_REQUEST"{t.Fatal(k)}}
 cases:=map[string]func(*policy.Bundle){
  "unsigned manifest":func(b *policy.Bundle){d:=fixture.Doc(b.Manifest);delete(d,"signature");b.Manifest=fixture.JSON(d)},
  "tampered manifest":func(b *policy.Bundle){d:=fixture.Doc(b.Manifest);d["body"].(signing.Object)["revision"]=99;b.Manifest=fixture.JSON(d)},
  "wrong site":func(b *policy.Bundle){d:=fixture.Doc(b.Job);d["site_id"]="99999999-9999-4999-8999-999999999999";b.Job=s.Sign(d,"mock-job")},
  "wrong purpose":func(b *policy.Bundle){d:=fixture.Doc(b.Job);d["signature"].(signing.Object)["kid"]="mock-grant";b.Job=fixture.JSON(d)},
  "missing consent":func(b *policy.Bundle){b.Grants=nil},
  "target mismatch":func(b *policy.Bundle){b.Grants[0]=fixture.Example("MC-09")},
  "missing backup":func(b *policy.Bundle){b.Backup=nil},
  "unapproved operation":func(b *policy.Bundle){*b=s.Job(1,false,nil,"mock-job")},
 }
 for name,modify:=range cases{t.Run(name,func(t *testing.T){b:=s.Job(1,true,[]string{"grant-a","grant-b"},"mock-job");modify(&b);if code:=h.apply(fixture.JSON(b),1001,1002);code!="OPERATION_DENIED"{t.Fatal(code)}})}
 // The independent verifier is unaffected by later mutation of the agent registry.
 delete(s.Config.Trust,"mock-job");if code:=h.apply(raw,1001,1002);code!="SIMULATED"{t.Fatal(code)}
 h.Close();h,err=Open(path,cfg,1001,1002,func()time.Time{return now});if err!=nil{t.Fatal(err)};defer h.Close()
 if code:=h.apply(raw,1001,1002);code!="REPLAY_DENIED"{t.Fatal(code)}
 now=now.AddDate(2,0,0);if code:=h.apply(raw,1001,1002);code!="OPERATION_DENIED"{t.Fatal(code)}
}
func TestUnapprovedArtifactAndLock(t *testing.T){
 for _,deny:=range []string{"artifact","clinical lock"}{t.Run(deny,func(t *testing.T){
 s:=fixture.NewSite();b:=s.Job(1,true,[]string{"a"},"mock-job");if deny=="artifact"{s.Config.TestArtifacts=map[string]string{"unrelated":"mock-secondary"}}else{s.Config.Preflight=func(string,[]string)bool{return false}}
 h,err:=Open(filepath.Join(t.TempDir(),"h.db"),s.Config,9,9,func()time.Time{return fixture.Now});if err!=nil{t.Fatal(err)};defer h.Close()
 if code:=h.apply(fixture.JSON(b),9,9);code!="OPERATION_DENIED"{t.Fatal(code)}
 })}
}
