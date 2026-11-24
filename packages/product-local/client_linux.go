//go:build linux

package productlocal
import("bytes";"context";"encoding/json";"errors";"io";"net";"net/http";"slices";"time";contracts "zyposoft.local/operations/packages/management-contracts";"zyposoft.local/operations/packages/unixpeer")
type Client struct{Endpoint unixpeer.Endpoint;WriteToken,HealthyCode string;KnownLocks []string}
func(c Client)call(ctx context.Context,method,path string,input any)([]byte,error){
 ctx,cancel:=context.WithTimeout(ctx,600*time.Millisecond);defer cancel()
 transport:=&http.Transport{DisableKeepAlives:true,MaxResponseHeaderBytes:4096,DialContext:func(ctx context.Context,_,_ string)(net.Conn,error){return c.Endpoint.Dial(ctx)}}
 client:=http.Client{Transport:transport,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 var body []byte;var err error;if input!=nil{body,err=json.Marshal(input);if err!=nil{return nil,err}}
 req,err:=http.NewRequestWithContext(ctx,method,"http://local"+path,bytes.NewReader(body));if err!=nil{return nil,err}
 if method=="POST"{req.Header.Set("Authorization","Bearer "+c.WriteToken)}
 resp,err:=client.Do(req);if err!=nil{return nil,err};defer resp.Body.Close();if resp.StatusCode!=200{return nil,errors.New("LOCAL_DENIED")}
 data,err:=io.ReadAll(io.LimitReader(resp.Body,8193));if err!=nil||len(data)>8192{return nil,errors.New("LOCAL_RESPONSE_LIMIT")}
 return contracts.Canonical(data)
}
func(c Client)read(ctx context.Context,path,shape string,dst any)error{
 wire,err:=c.call(ctx,"GET","/local/v1/"+path,nil);if err!=nil{return err}
 if _,_,err=contracts.Decode("LOCAL-01/"+shape,wire);err!=nil{return err};return json.Unmarshal(wire,dst)
}
func(c Client)Preflight(ctx context.Context,locks []string)error{
 for _,lock:=range locks{if !slices.Contains(c.KnownLocks,lock){return errors.New("UNKNOWN_LOCK")}}
 var l contracts.LocksResponse;if err:=c.read(ctx,"locks","locks_response",&l);err!=nil{return err}
 if l.Locked||len(l.Reasons)!=0||l.LeaseTtlS<=0{return errors.New("SITE_LOCK_ACTIVE")}
 ok,err:=c.ReadyVersion(ctx,"");if err!=nil{return err};if !ok{return errors.New("PRODUCT_UNREADY")};return nil
}
func(c Client)ReadyVersion(ctx context.Context,version string)(bool,error){var r contracts.ReadinessResponse;err:=c.read(ctx,"readiness","readiness_response",&r);return r.Ready&&slices.Contains(r.ServiceCodes,c.HealthyCode)&&(version==""||r.Version==version),err}
func(c Client)effect(ctx context.Context,path string,a Activation)(Observation,error){
 var o Observation;data,err:=c.call(ctx,"POST","/simulation/"+path,a);if err!=nil{return o,err};err=json.Unmarshal(data,&o)
 if err!=nil||o.Activation!=a{return o,errors.New("PRODUCT_BINDING")};return o,nil
}
func(c Client)Apply(ctx context.Context,a Activation)(Observation,error){return c.effect(ctx,"apply",a)}
func(c Client)Lookup(ctx context.Context,a Activation)(Observation,error){return c.effect(ctx,"lookup",a)}

func(c Client)Fence(ctx context.Context,a Activation)(Observation,error){return c.effect(ctx,"fence",a)}
