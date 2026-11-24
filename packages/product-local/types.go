// Package productlocal contains test-local effect messages alongside the unchanged LOCAL-01 read interfaces.
package productlocal
import("context";"errors")
type Activation struct{Product,Job,Execution,Artifact,Release,Expires string}
type Observation struct{Activation; Phase string}
type Application interface{
 Preflight(context.Context,[]string)error
 Lookup(context.Context,Activation)(Observation,error)
 Fence(context.Context,Activation)(Observation,error)
 Apply(context.Context,Activation)(Observation,error)
 ReadyVersion(context.Context,string)(bool,error)
}
type Registry map[string]Application
func(r Registry)Check(ctx context.Context,key string,locks []string)error{app:=r[key];if app==nil{return errors.New("UNKNOWN_PRODUCT")};return app.Preflight(ctx,locks)}
