//go:build linux

package sprint2

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"zyposoft.local/operations/packages/deployment"
	contracts "zyposoft.local/operations/packages/management-contracts"
	"zyposoft.local/operations/packages/unixpeer"
	"zyposoft.local/operations/tests/fixtures"
)

func TestInvalidHelperFrames(t *testing.T) {
	dir := testDir(t)
	p := makePlan(dir, true)
	startApps(t, dir, p)
	s := serveHelper(t, dir, p)
	validBundle := string(fixture.JSON(p.jobs[0]))
	for _, tc := range []struct {
		name, body string
		size       uint32
		want       string
	}{
		{"zero", "", 0, "INVALID_REQUEST"}, {"oversized", "", contracts.MaxBytes + 1, "INVALID_REQUEST"},
		{"truncated", "{}", 100, ""}, {"malformed", "{", 1, "DENIED"},
		{"duplicate", `{"operation":"lookup","operation":"execute","bundle":` + validBundle + "}", 0, "DENIED"},
		{"unknown", `{"operation":"execute","privileges":["x"],"bundle":` + validBundle + "}", 0, "DENIED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: s.Endpoint.Path, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			size := tc.size
			if size == 0 {
				size = uint32(len(tc.body))
			}
			if err = c.SetDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			if err = binary.Write(c, binary.BigEndian, size); err != nil {
				t.Fatal(err)
			}
			if _, err = io.WriteString(c, tc.body); err != nil {
				t.Fatal(err)
			}
			if err = c.CloseWrite(); err != nil {
				t.Fatal(err)
			}
			line, err := bufio.NewReader(c).ReadString('\n')
			if tc.want == "" {
				if err != io.EOF || line != "" {
					t.Fatal(line, err)
				}
			} else if err != nil || !strings.Contains(line, tc.want) {
				t.Fatal(line, err)
			}
		})
	}
	if n := scalar(t, filepath.Join(dir, "helper.db"), "SELECT count(*) FROM consumed_grant"); n != 0 {
		t.Fatal(n)
	}
	if n := scalar(t, filepath.Join(dir, p.profiles[0].Product+".db"), "SELECT count(*) FROM effect"); n != 0 {
		t.Fatal(n)
	}
}
func TestInvalidHelperReplies(t *testing.T) {
	for _, tc := range []struct{ name, line string }{
		{"oversize", strings.Repeat(" ", 4096) + "{}\n"}, {"truncated", `{"status":"ABSENT"}`}, {"duplicate", `{"status":"ABSENT","status":"REFUSED"}` + "\n"},
		{"unknown field", `{"status":"ABSENT","root":true}` + "\n"}, {"unknown status", `{"status":"OK"}` + "\n"}, {"unexpected result", `{"status":"ABSENT","result":{}}` + "\n"},
		{"missing result", `{"status":"DONE"}` + "\n"}, {"timeout", ""}, {"wrong server UID", ""}, {"wrong server GID", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(testDir(t), "wire.sock")
			l, err := unixpeer.Listen(path)
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			done := make(chan struct{})
			go func() {
				defer close(done)
				c, e := l.AcceptUnix()
				if e != nil {
					return
				}
				defer c.Close()
				var n uint32
				if binary.Read(c, binary.BigEndian, &n) != nil || n > contracts.MaxBytes {
					return
				}
				if _, e = io.CopyN(io.Discard, c, int64(n)); e != nil {
					return
				}
				if tc.name == "timeout" {
					<-ctx.Done()
					return
				}
				io.WriteString(c, tc.line)
			}()
			ep := endpoint(path)
			if tc.name == "wrong server UID" {
				ep.Peer.UID++
			}
			if tc.name == "wrong server GID" {
				ep.Peer.GID++
			}
			if _, e := (deployment.Socket{Endpoint: ep}).Exchange(ctx, deployment.Request{Operation: "lookup"}); e == nil {
				t.Fatal("invalid reply accepted")
			}
			<-done
		})
	}
}
func TestSocketPathSafety(t *testing.T) {
	dir := testDir(t)
	path := filepath.Join(dir, "socket")
	if err := os.WriteFile(path, []byte("owner data"), 0600); err != nil {
		t.Fatal(err)
	}
	if l, err := unixpeer.Listen(path); err == nil {
		l.Close()
		t.Fatal("replaced regular file")
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != "owner data" {
		t.Fatal(err)
	}
	unsafe := testDir(t)
	if err := os.Chmod(unsafe, 0777); err != nil {
		t.Fatal(err)
	}
	if l, err := unixpeer.Listen(filepath.Join(unsafe, "zop-test.sock")); err == nil {
		l.Close()
		t.Fatal("writable directory")
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(path, alias+".lock"); err != nil {
		t.Fatal(err)
	}
	if other, e := unixpeer.Listen(alias); e == nil {
		other.Close()
		t.Fatal("followed lock symlink")
	}
	active := filepath.Join(dir, "active")
	l, err := unixpeer.Listen(active)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if other, e := unixpeer.Listen(active); e == nil {
		other.Close()
		t.Fatal("replaced active peer")
	}
}
