package signing

import (
 "crypto"
 "crypto/ecdsa"
 "crypto/ed25519"
 "crypto/elliptic"
 "crypto/rand"
 "crypto/sha256"
 "encoding/base64"
 "encoding/json"
 "errors"
 "maps"
 "math/big"
 "strings"
 contracts "zyposoft.local/operations/packages/management-contracts"
)

type Scope struct {Tenant,Site string}
type Key struct {Alg,Purpose string; Scope Scope; Public crypto.PublicKey; Revoked bool}
type Trust map[string]Key
type Object = map[string]any

func input(domain string,obj Object)([]byte,error){
 raw,err:=json.Marshal(obj);if err!=nil{return nil,err}
 data,err:=contracts.Canonical(raw);if err!=nil{return nil,err}
 return append([]byte(domain+"\x00"),data...),nil
}
func EnvelopeBytes(doc Object)([]byte,error){
 unsigned:=maps.Clone(doc);delete(unsigned,"signature")
 return input("ZOP-SIG-V1\x00"+doc["contract_id"].(string),unsigned)
}
func nested(doc Object)(field,purpose string,data []byte,err error){
 body:=maps.Clone(doc["body"].(map[string]any))
 switch doc["contract_id"]{
 case "MC-09":
  if body["maker_id"]==body["checker_id"]{err=errors.New("MAKER_CHECKER_SAME");return}
  field,purpose="site_authoriser_attestation","site_authoriser";delete(body,field)
  data,err=input("ZOP-SITE-CONSENT-V1",Object{"tenant_id":doc["tenant_id"],"site_id":doc["site_id"],"body":body})
 case "MC-12":
  field,purpose="clinical_owner_signature","clinical_owner";delete(body,field)
  data,err=input("ZOP-CLINICAL-PACKAGE-V1",body)
 }
 return
}
func keyFor(trust Trust,kid,alg,purpose string,doc Object)(Key,error){
 k,ok:=trust[kid]
 if !ok||k.Revoked||k.Alg!=alg||k.Purpose!=purpose{return Key{},errors.New("UNTRUSTED_KEY")}
 if (strings.HasPrefix(purpose,"agent_")||purpose=="site_authoriser")&&(k.Scope.Tenant==""||k.Scope.Site==""){return Key{},errors.New("UNSCOPED_KEY")}
 if (k.Scope.Tenant!=""&&doc["tenant_id"]!=k.Scope.Tenant)||(k.Scope.Site!=""&&doc["site_id"]!=k.Scope.Site){return Key{},errors.New("KEY_SCOPE")}
 return k,nil
}
func check(doc Object,trust Trust,signature Object,purpose string,msg []byte)error{
 k,err:=keyFor(trust,signature["kid"].(string),signature["alg"].(string),purpose,doc);if err!=nil{return err}
 value:=signature["value"].(string)
 sig,err:=base64.RawURLEncoding.Strict().DecodeString(value)
 if err!=nil||len(sig)!=64||base64.RawURLEncoding.EncodeToString(sig)!=value{return errors.New("SIGNATURE_FORMAT")}
 switch pub:=k.Public.(type){
 case ed25519.PublicKey:
  if k.Alg=="Ed25519" && len(pub)==32 && ed25519.Verify(pub,msg,sig){return nil}
 case *ecdsa.PublicKey:
  digest:=sha256.Sum256(msg)
  if k.Alg=="ES256" && pub.Curve==elliptic.P256() && ecdsa.Verify(pub,digest[:],new(big.Int).SetBytes(sig[:32]),new(big.Int).SetBytes(sig[32:])){return nil}
 }
 return errors.New("BAD_SIGNATURE")
}
func Verify(id string,raw []byte,scope Scope,trust Trust)(Object,error){
 doc,entry,err:=contracts.Decode(id,raw);if err!=nil{return nil,err}
 if entry.ID=="LOCAL-01"{return nil,errors.New("UNSIGNED_INTERFACE")}
 if entry.Scope=="site_scoped" && (doc["tenant_id"]!=scope.Tenant||doc["site_id"]!=scope.Site){return nil,errors.New("MESSAGE_SCOPE")}
 data,err:=EnvelopeBytes(doc);if err!=nil{return nil,err}
 if err=check(doc,trust,doc["signature"].(Object),entry.Key_Purpose,data);err!=nil{return nil,err}
 field,purpose,data,err:=nested(doc);if err!=nil{return nil,err}
 if field!=""{if err=check(doc,trust,doc["body"].(Object)[field].(Object),purpose,data);err!=nil{return nil,err}}
 return doc,nil
}
// Sign validates the complete candidate and re-verifies the resulting envelope under the supplied policy.
func Sign(id string,unsigned []byte,scope Scope,trust Trust,kid string,private crypto.PrivateKey)([]byte,error){
 data,err:=contracts.Canonical(unsigned);if err!=nil{return nil,err}
 var doc Object
 if err=json.Unmarshal(data,&doc);err!=nil||doc==nil{return nil,errors.New("SIGNING_INPUT")}
 if _,present:=doc["signature"];present||doc["contract_id"]!=id{return nil,errors.New("SIGNING_INPUT")}
 key:=trust[kid]
 sig:=Object{"kid":kid,"alg":key.Alg,"value":strings.Repeat("A",86)};doc["signature"]=sig
 candidate,err:=json.Marshal(doc);if err!=nil{return nil,err}
 _,entry,err:=contracts.Decode(id,candidate);if err!=nil{return nil,err}
 if _,err=keyFor(trust,kid,key.Alg,entry.Key_Purpose,doc);err!=nil{return nil,err}
 msg,err:=EnvelopeBytes(doc);if err!=nil{return nil,err}
 raw,err:=signRaw(key,private,msg);if err!=nil{return nil,err}
 sig["value"]=base64.RawURLEncoding.EncodeToString(raw)
 result,err:=json.Marshal(doc);if err!=nil{return nil,err}
 _,err=Verify(id,result,scope,trust);return result,err
}
func signRaw(key Key,private crypto.PrivateKey,msg []byte)([]byte,error){
 var raw []byte
 switch priv:=private.(type){
 case ed25519.PrivateKey:
  if len(priv)!=ed25519.PrivateKeySize||key.Alg!="Ed25519"{return nil,errors.New("KEY_TYPE")}
  raw=ed25519.Sign(priv,msg)
 case *ecdsa.PrivateKey:
  if priv.Curve!=elliptic.P256()||key.Alg!="ES256"{return nil,errors.New("KEY_TYPE")}
  digest:=sha256.Sum256(msg);r,s,err:=ecdsa.Sign(rand.Reader,priv,digest[:]);if err!=nil{return nil,err}
  raw=make([]byte,64);r.FillBytes(raw[:32]);s.FillBytes(raw[32:])
 default:return nil,errors.New("KEY_TYPE")
 }
 return raw,nil
}
// SignAttestation signs an MC-09 or MC-12 body before the outer issuer signs.
func SignAttestation(id string,unsigned []byte,trust Trust,kid string,private crypto.PrivateKey)(Object,error){
 if id!="MC-09"&&id!="MC-12"{return nil,errors.New("ATTESTATION_TYPE")}
 data,err:=contracts.Canonical(unsigned);if err!=nil{return nil,err}
 var doc Object;if err=json.Unmarshal(data,&doc);err!=nil||doc["contract_id"]!=id{return nil,errors.New("SIGNING_INPUT")}
 body,ok:=doc["body"].(Object);if !ok{return nil,errors.New("INVALID_BODY")}
 field:="site_authoriser_attestation";if id=="MC-12"{field="clinical_owner_signature"}
 key:=trust[kid];sig:=Object{"kid":kid,"alg":key.Alg,"value":strings.Repeat("A",86)}
 body[field]=sig;doc["signature"]=Object{"kid":"placeholder","alg":"Ed25519","value":strings.Repeat("A",86)}
 candidate,err:=json.Marshal(doc);if err!=nil{return nil,err};if _,_,err=contracts.Decode(id,candidate);err!=nil{return nil,err}
 _,purpose,msg,err:=nested(doc);if err!=nil{return nil,err}
 if _,err=keyFor(trust,kid,key.Alg,purpose,doc);err!=nil{return nil,err}
 raw,err:=signRaw(key,private,msg);if err!=nil{return nil,err};sig["value"]=base64.RawURLEncoding.EncodeToString(raw)
 return sig,check(doc,trust,sig,purpose,msg)
}
