// Package fixture loads only synthetic public test material for regression suites.
package fixture
import("crypto/x509";"encoding/pem";"encoding/json";"embed";"strings";"zyposoft.local/operations/packages/signing-go")
var Scope=signing.Scope{Tenant:"11111111-1111-4111-8111-111111111111",Site:"22222222-2222-4222-8222-222222222222"}
//go:embed examples/*.json vectors/*.json vectors/public_keys/*.pem canonical.json test_key_registry.json
var files embed.FS
func Read(path string)[]byte{b,err:=files.ReadFile(path);if err!=nil{panic(err)};return b}
func Example(id string)[]byte{return Read("examples/"+id+".example.json")}
func Trust()signing.Trust{
 var registry map[string]struct{Alg,Purpose,Public_key_file,Tenant_id,Site_id string}
 if err:=json.Unmarshal(Read("test_key_registry.json"),&registry);err!=nil{panic(err)}
 keys:=signing.Trust{}
 for kid,rec:=range registry{
  block,_:=pem.Decode(Read(rec.Public_key_file));pub,err:=x509.ParsePKIXPublicKey(block.Bytes);if err!=nil{panic(err)}
  scope:=signing.Scope{Tenant:rec.Tenant_id,Site:rec.Site_id};if strings.HasPrefix(rec.Purpose,"agent_"){scope=Scope}
  keys[kid]=signing.Key{Public:pub,Purpose:rec.Purpose,Alg:rec.Alg,Scope:scope}
 };return keys
}
