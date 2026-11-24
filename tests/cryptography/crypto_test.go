package cryptography
import("encoding/base64";"encoding/json";"strings";"testing";contracts "zyposoft.local/operations/packages/management-contracts";"zyposoft.local/operations/packages/signing-go";"zyposoft.local/operations/tests/fixtures")
func TestCorpus(t *testing.T){
 var corpus struct{Valid [][]string;Invalid []string};if err:=json.Unmarshal(fixture.Read("canonical.json"),&corpus);err!=nil{t.Fatal(err)}
 for _,v:=range corpus.Valid{got,err:=contracts.Canonical([]byte(v[0]));if err!=nil||string(got)!=v[1]{t.Fatalf("%s: %s %v",v[0],got,err)}}
 for _,bad:=range append(corpus.Invalid,strings.Repeat("[",65)+strings.Repeat("]",65),strings.Repeat(" ",contracts.MaxBytes+1),string([]byte{34,255,34})){if _,err:=contracts.Canonical([]byte(bad));err==nil{t.Fatal("invalid input accepted")}}
}
func TestGolden(t *testing.T){
 var vectors []struct{Contract_id,Input_b64 string};if err:=json.Unmarshal(fixture.Read("vectors/signed_vectors.json"),&vectors);err!=nil{t.Fatal(err)}
 keys:=fixture.Trust()
 for _,v:=range vectors{t.Run(v.Contract_id,func(t *testing.T){
 doc,err:=signing.Verify(v.Contract_id,fixture.Example(v.Contract_id),fixture.Scope,keys);if err!=nil{t.Fatal(err)}
 data,err:=signing.EnvelopeBytes(doc);if err!=nil||base64.StdEncoding.EncodeToString(data)!=v.Input_b64{t.Fatal("canonical mismatch")}
 })}
}
func TestNegative(t *testing.T){
 var vectors []struct{Message json.RawMessage};if err:=json.Unmarshal(fixture.Read("vectors/negative_signed_vectors.json"),&vectors);err!=nil{t.Fatal(err)}
 for _,v:=range vectors{var h struct{Contract_id,Tenant_id,Site_id string};if err:=json.Unmarshal(v.Message,&h);err!=nil{t.Fatal(err)};if _,err:=signing.Verify(h.Contract_id,v.Message,signing.Scope{Tenant:h.Tenant_id,Site:h.Site_id},fixture.Trust());err==nil{t.Fatal("accepted invalid signature")}}
}
