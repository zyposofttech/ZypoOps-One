//go:build linux

package deployment
import("bufio";"bytes";"context";"encoding/binary";"encoding/json";"errors";"time";contracts "zyposoft.local/operations/packages/management-contracts";"zyposoft.local/operations/packages/unixpeer")
type Socket struct{Endpoint unixpeer.Endpoint}
func(s Socket)Exchange(ctx context.Context,req Request)(Reply,error){
 var reply Reply
 data,err:=json.Marshal(req);if err!=nil||len(data)>contracts.MaxBytes{return reply,errors.New("REQUEST_LIMIT")}
 ctx,cancel:=context.WithTimeout(ctx,4*time.Second);defer cancel()
 conn,err:=s.Endpoint.Dial(ctx);if err!=nil{return reply,err};defer conn.Close();stop:=context.AfterFunc(ctx,func(){conn.Close()});defer stop()
 deadline,_:=ctx.Deadline();if err=conn.SetDeadline(deadline);err!=nil{return reply,err}
 if err=binary.Write(conn,binary.BigEndian,uint32(len(data)));err!=nil{return reply,err}
 if _,err=conn.Write(data);err!=nil{return reply,err}
 line,err:=bufio.NewReaderSize(conn,4096).ReadSlice('\n');if err!=nil{return reply,err}
 line,err=contracts.Canonical(line);if err!=nil{return reply,err}
 decoder:=json.NewDecoder(bytes.NewReader(line));decoder.DisallowUnknownFields();if err=decoder.Decode(&reply);err!=nil{return reply,err}
 if reply.Status=="DONE" {if reply.Result==nil{return reply,errors.New("MISSING_RESULT")};err=reply.Result.Validate(req.Bundle)}
 return reply,err
}
