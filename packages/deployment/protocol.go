// Package deployment defines the bounded, test-local helper interface. Management envelopes remain O0 v1.2.
package deployment
import("context";"encoding/json";"errors";"regexp";"github.com/google/uuid";"zyposoft.local/operations/packages/local-policy";contracts "zyposoft.local/operations/packages/management-contracts")
type Request struct {Operation string `json:"operation"`; Bundle policy.Bundle `json:"bundle"`}
type Result struct {Job string `json:"job"`; Execution string `json:"execution"`; JobDigest string `json:"job_digest"`; Artifact string `json:"artifact"`; Product string `json:"product"`; State string `json:"state"`; Stage string `json:"stage"`; Code string `json:"code"`}
type Reply struct{Status string `json:"status"`; Result *Result `json:"result,omitempty"`}
type Helper interface{Exchange(context.Context,Request)(Reply,error)}
func BundleDigest(b policy.Bundle)(string,error){wire,err:=json.Marshal(b);if err!=nil{return "",err};return policy.Digest(wire)}
var digestPattern=regexp.MustCompile("^[0-9a-f]{64}$")
func(r Result)Validate(b policy.Bundle)error{
 var j contracts.MC03;var p contracts.DeployReleaseParams
 digest,err:=policy.Digest(b.Job)
 if err!=nil||json.Unmarshal(b.Job,&j)!=nil||json.Unmarshal(j.Body.Params,&p)!=nil||r.Job!=j.Body.JobId||r.JobDigest!=digest||r.Product!=p.ProductKey||r.Artifact!=p.ManifestSha256||!digestPattern.MatchString(r.Artifact){return errors.New("RESULT_BINDING")}
 if _,err=uuid.Parse(r.Execution);err!=nil{return errors.New("EXECUTION_ID")}
 switch r.Stage{case "VERIFIED":if r.State=="SUCCEEDED"&&r.Code=="NONE"{return nil}
 case "BEFORE_EFFECT":if r.State=="REJECTED"&&r.Code=="PRECONDITION_FAILED"{return nil}
 case "POST_VERIFY":if r.State=="FAILED"&&r.Code=="PRECONDITION_FAILED"||r.State=="UNKNOWN"&&r.Code=="EXECUTION_UNKNOWN"{return nil}
 case "AFTER_EFFECT":if r.State=="FAILED"&&r.Code=="OTHER_CODED"{return nil}}
 return errors.New("INVALID_OUTCOME")
}
