//go:build linux

package helper
import("bufio";"context";"encoding/binary";"errors";"io";"net";"os";"path/filepath";"syscall";"time";"golang.org/x/sys/unix";contracts "zyposoft.local/operations/packages/management-contracts")
// Each connection carries one big-endian length-prefixed JSON request and receives one bounded status line.
func(h *Helper)Serve(ctx context.Context,path string)error{
 dir,err:=os.Lstat(filepath.Dir(path));if err!=nil{return err}
 owner,ok:=dir.Sys().(*syscall.Stat_t);if !ok||!dir.IsDir()||dir.Mode().Perm()&0022!=0||owner.Uid!=uint32(os.Geteuid()){return errors.New("UNSAFE_SOCKET_DIRECTORY")}
 listener,err:=net.ListenUnix("unix",&net.UnixAddr{Name:path,Net:"unix"});if err!=nil{return err};defer listener.Close()
 if err=os.Chmod(path,0660);err!=nil{return err}
 stop:=context.AfterFunc(ctx,func(){listener.Close()});defer stop()
 for{conn,err:=listener.AcceptUnix();if err!=nil{if ctx.Err()!=nil{return nil};return err};h.handleConn(conn)}
}
func(h *Helper)handleConn(conn *net.UnixConn){
 defer conn.Close();if conn.SetDeadline(time.Now().Add(2*time.Second))!=nil{return}
 code:="UNAUTHORISED_PEER"
 descriptor,err:=conn.SyscallConn();var peer *unix.Ucred;var socketErr error
 if err==nil{err=descriptor.Control(func(fd uintptr){peer,socketErr=unix.GetsockoptUcred(int(fd),unix.SOL_SOCKET,unix.SO_PEERCRED)})}
 if err==nil&&socketErr==nil&&peer!=nil&&peer.Uid==h.uid&&peer.Gid==h.gid{
  code="INVALID_REQUEST";reader:=bufio.NewReader(conn);var size uint32
  if err=binary.Read(reader,binary.BigEndian,&size);err==nil&&size>0&&size<=contracts.MaxBytes{raw:=make([]byte,size);if _,err=io.ReadFull(reader,raw);err==nil{code=h.apply(raw,peer.Uid,peer.Gid)}}
 }
 _,_=io.WriteString(conn,code+"\n")
}
