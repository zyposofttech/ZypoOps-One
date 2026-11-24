package contracts

import (
 "embed"
 "bytes"
 "io"
 "encoding/json"
 "errors"
 "sync"
 "github.com/go-json-experiment/json/jsontext"
 schema "github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/*.json catalog.json
var source embed.FS
type Entry struct {ID, Scope, Schema, Key_Purpose string}
type descriptor struct {Entry; validator *schema.Schema}
var registry = sync.OnceValues(compile)

func compile()(map[string]descriptor,error){
 var catalog []Entry
 data,err:=source.ReadFile("catalog.json");if err!=nil{return nil,err}
 if err=json.Unmarshal(data,&catalog);err!=nil{return nil,err}
 validators:=make(map[string]descriptor)
 compiler:=schema.NewCompiler();compiler.AssertFormat()
 for _,entry:=range append(catalog,Entry{ID:"LOCAL-01", Schema:"schemas/LOCAL-01.product-management.schema.json"}){
  raw,err:=source.ReadFile(entry.Schema);if err!=nil{return nil,err}
  var def map[string]any
  if err=json.Unmarshal(raw,&def);err!=nil{return nil,err}
  uri:=def["$id"].(string)
  if err=compiler.AddResource(uri,def);err!=nil{return nil,err}
  compiled,err:=compiler.Compile(uri);if err!=nil{return nil,err}
  validators[entry.ID]=descriptor{entry,compiled}
  if entry.ID=="LOCAL-01"{for k:=range def["$defs"].(map[string]any){ v,err:=compiler.Compile(uri+"#/$defs/"+k);if err!=nil{return nil,err};validators["LOCAL-01/"+k]=descriptor{entry,v} }}
 }
 return validators,nil
}
const MaxBytes=1<<20
func Canonical(raw []byte)([]byte,error){
 if len(raw)>MaxBytes{return nil,errors.New("PAYLOAD_LIMIT")}
 // encoding/json rejects numeric overflow before JCS could round it to a finite limit.
 var checked any
 value:=jsontext.Value(raw).Clone()
 decoder:=jsontext.NewDecoder(bytes.NewReader(raw))
 for {_,err:=decoder.ReadToken();if err==io.EOF{break};if err!=nil||decoder.StackDepth()>64{return nil,errors.New("INVALID_JSON")}}
 if err:=json.Unmarshal(raw,&checked);err!=nil{return nil,errors.New("INVALID_NUMBER")}
 if err:=value.Canonicalize();err!=nil{return nil,errors.New("INVALID_JSON")}
 return value,nil
}
func Decode(id string,raw []byte)(map[string]any,Entry,error){
 validators,err:=registry();if err!=nil{return nil,Entry{},err}
 entry,ok:=validators[id];if !ok{return nil,Entry{},errors.New("UNKNOWN_CONTRACT")}
 data,err:=Canonical(raw);if err!=nil{return nil,Entry{},err}
 var doc map[string]any
 if err=json.Unmarshal(data,&doc);err!=nil{return nil,Entry{},errors.New("INVALID_OBJECT")}
 if entry.ID!="LOCAL-01" && doc["schema_version"]!="1.2.0"{return nil,Entry{},errors.New("UNKNOWN_VERSION")}
 if err=entry.validator.Validate(doc);err!=nil{return nil,Entry{},errors.New("INVALID_CONTRACT")}
 return doc,entry.Entry,nil
}
