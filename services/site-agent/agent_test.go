package agent
import("bytes";"context";"errors";"os";"os/exec";"path/filepath";"strings";"testing";"time";"zyposoft.local/operations/tests/fixtures";"zyposoft.local/operations/packages/local-policy";"zyposoft.local/operations/packages/signing-go")
func openAgent(t *testing.T,path string,s *fixture.Site)*Agent{t.Helper();a,err:=Open(path,Identity{fixture.Scope.Tenant,fixture.Scope.Site,fixture.Install,fixture.AgentID,strings.Repeat("a",64)},s.Config,s,nil);if err!=nil{t.Fatal(err)};return a}
func sample()Sample{return Sample{fixture.Now,"boot-A",10,true}}
func TestSequenceClockAndConsent(t *testing.T){
 s:=fixture.NewSite();path:=filepath.Join(t.TempDir(),"ledger.sqlite");a:=openAgent(t,path,s);defer a.Close();ctx:=context.Background()
 var mode string;if err:=a.db.QueryRow("PRAGMA journal_mode").Scan(&mode);err!=nil||mode!="wal"{t.Fatal(mode,err)}
 job:=s.Job(1,true,[]string{"first","second"},"mock-job");first,err:=a.Handle(ctx,job,sample());if err!=nil{t.Fatal(err)}
 replay,err:=a.Handle(ctx,job,sample());if err!=nil||!bytes.Equal(first,replay){t.Fatal("duplicate changed receipt",err)}
 if _,err=signing.Verify("MC-04",first,fixture.Scope,s.Config.Trust);err!=nil{t.Fatal(err)}
 negatives:=[]policy.Bundle{s.Job(1,false,nil,"rotated-job"),s.Job(3,false,nil,"rotated-job"),s.Job(2,true,[]string{"first"},"mock-job")}
 wrong:=s.Job(2,false,nil,"mock-job");d:=fixture.Doc(wrong.Job);d["site_id"]="99999999-9999-4999-8999-999999999999";wrong.Job=s.Sign(d,"mock-job");negatives=append(negatives,wrong)
 for _,bad:=range negatives{if _,err=a.Handle(ctx,bad,sample());err==nil{t.Fatal("invalid accepted")}}
 next:=s.Job(2,false,nil,"rotated-job");if _,err=a.Handle(ctx,next,sample());err!=nil{t.Fatal(err)}
 backwards:=sample();backwards.UTC=backwards.UTC.Add(-time.Hour);if _,err=a.Handle(ctx,s.Job(3,false,nil,"mock-job"),backwards);err==nil{t.Fatal("clock rollback accepted")}
 a.Close();a=openAgent(t,path,s);defer a.Close()
 if _,err=a.Handle(ctx,s.Job(3,false,nil,"mock-job"),sample());err==nil{t.Fatal("clock confidence lost on restart")}
 for table,want:=range map[string]int{"received_job":2,"job_stage":2,"execution_receipt":2,"approval_consumption":2}{var n int;if err=a.db.QueryRow("SELECT count(*) FROM "+table).Scan(&n);err!=nil||n!=want{t.Fatal(table,n,err)}}
}
type mockTransport struct{next policy.Bundle;fail bool;sent int}
func(m *mockTransport)Next(context.Context)(policy.Bundle,error){if m.next.Job==nil{return policy.Bundle{},errors.New("OFFLINE")};b:=m.next;m.next=policy.Bundle{};return b,nil}
func(m *mockTransport)Receipt(_ context.Context,_ []byte)error{if m.fail{return errors.New("OFFLINE")};m.sent++;return nil}
func TestDisconnectedReceiptRecovery(t *testing.T){
 s:=fixture.NewSite();path:=filepath.Join(t.TempDir(),"agent.db");a:=openAgent(t,path,s);m:=mockTransport{next:s.Job(1,false,nil,"mock-job"),fail:true}
 if err:=a.Run(context.Background(),&m,sample);err==nil{t.Fatal("expected outage")};a.Close()
 a=openAgent(t,path,s);defer a.Close();m.fail=false
 if err:=a.Run(context.Background(),&m,sample);err==nil||m.sent!=1{t.Fatal("pending receipt not delivered",m.sent,err)}
}
func TestFaultRecovery100(t *testing.T){
 s:=fixture.NewSite();path:=filepath.Join(t.TempDir(),"agent.db");ctx:=context.Background();a:=openAgent(t,path,s)
 for i:=1;i<=100;i++{
  job:=s.Job(i,false,nil,"mock-job");a.fault=func(string)error{return errors.New("SIMULATED_CRASH")}
  if _,err:=a.Handle(ctx,job,sample());err==nil{t.Fatal(i,"fault missed")};a.Close();a=openAgent(t,path,s)
  if _,err:=a.Handle(ctx,job,sample());err!=nil{t.Fatal(i,err)}
 };defer a.Close();var n int;if err:=a.db.QueryRow("SELECT count(*) FROM job_stage").Scan(&n);err!=nil||n!=100{t.Fatal(n,err)}
}
func TestProcessDeath(t *testing.T){
 s:=fixture.NewSite();job:=s.Job(1,false,nil,"mock-job");path:=os.Getenv("ZOP_FAULT_DB")
 if path!=""{a:=openAgent(t,path,s);a.fault=func(string)error{os.Exit(71);return nil};_,_=a.Handle(context.Background(),job,sample());t.Fatal("fault missed")}
 path=filepath.Join(t.TempDir(),"agent.db")
 cmd:=exec.Command(os.Args[0],"-test.run=^TestProcessDeath$");cmd.Env=append(os.Environ(),"ZOP_FAULT_DB="+path);err:=cmd.Run();var exit *exec.ExitError;if !errors.As(err,&exit)||exit.ExitCode()!=71{t.Fatal(err)}
 a:=openAgent(t,path,s);defer a.Close();if _,err=a.Handle(context.Background(),job,sample());err!=nil{t.Fatal(err)}
 var n int;if err=a.db.QueryRow("SELECT count(*) FROM job_stage").Scan(&n);err!=nil||n!=1{t.Fatal(n,err)}
}
