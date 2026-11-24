//go:build linux

package helper

import (
	"bufio"
	"context"
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
	"zyposoft.local/operations/tests/fixtures"
)

func TestKernelSocketPeer(t *testing.T) {
	for _, name := range []string{"allowed", "wrong UID", "wrong GID"} {
		deny := name != "allowed"
		t.Run(name, func(t *testing.T) {
			s := fixture.NewSite()
			uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
			if name == "wrong UID" {
				uid++
			}
			if name == "wrong GID" {
				gid++
			}
			dir := t.TempDir()
			path := filepath.Join(dir, "mock.sock")
			h, err := Open(filepath.Join(dir, "helper.db"), s.Config, uid, gid, func() time.Time { return fixture.Now })
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- h.Serve(ctx, path) }()
			defer func() {
				cancel()
				if err := <-done; err != nil {
					t.Error(err)
				}
			}()
			var conn net.Conn
			for i := 0; i < 100; i++ {
				conn, err = net.Dial("unix", path)
				if err == nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			want := "UNAUTHORISED_PEER\n"
			if !deny {
				data := fixture.JSON(s.Job(1, true, []string{"grant"}, "mock-job"))
				if err = binary.Write(conn, binary.BigEndian, uint32(len(data))); err != nil {
					t.Fatal(err)
				}
				if _, err = conn.Write(data); err != nil {
					t.Fatal(err)
				}
				want = "SIMULATED\n"
			}
			got, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil || got != want {
				t.Fatal(got, err)
			}
		})
	}
}
