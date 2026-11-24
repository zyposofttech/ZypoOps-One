// Test-only one-shot interoperability witness. Keys live for one process.
package main
import("crypto";"crypto/ecdsa";"crypto/ed25519";"crypto/elliptic";"crypto/rand";"crypto/x509";"encoding/base64";"encoding/json";"io";"os";"zyposoft.local/operations/packages/signing-go";"zyposoft.local/operations/tests/fixtures")
func main(){
 var req struct{Alg,Public string; Message json.RawMessage}
 if err:=json.NewDecoder(io.LimitReader(os.Stdin,1<<20)).Decode(&req);err!=nil{panic(err)}
 der,err:=base64.StdEncoding.DecodeString(req.Public);if err!=nil{panic(err)}
 pub,err:=x509.ParsePKIXPublicKey(der);if err!=nil{panic(err)}
 key:=signing.Key{Alg:req.Alg,Purpose:"agent_receipt",Scope:fixture.Scope,Public:pub}
 keys:=signing.Trust{"ts-test":key}
 doc,err:=signing.Verify("MC-04",req.Message,fixture.Scope,keys);if err!=nil{panic(err)}
 var priv crypto.Signer
 if req.Alg=="ES256"{priv,err=ecdsa.GenerateKey(elliptic.P256(),rand.Reader)}else{_,priv,err=ed25519.GenerateKey(rand.Reader)};if err!=nil{panic(err)}
 key.Public=priv.Public();keys["go-test"]=key
 delete(doc,"signature");raw,err:=json.Marshal(doc);if err!=nil{panic(err)}
 signed,err:=signing.Sign("MC-04",raw,fixture.Scope,keys,"go-test",priv);if err!=nil{panic(err)}
 der,err=x509.MarshalPKIXPublicKey(priv.Public());if err!=nil{panic(err)}
 if err=json.NewEncoder(os.Stdout).Encode(map[string]any{"public":base64.StdEncoding.EncodeToString(der),"message":json.RawMessage(signed)});err!=nil{panic(err)}
}
