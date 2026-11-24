package agent

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"zyposoft.local/operations/tests/fixtures"
)

// The parent kills the agent while a write transaction is open and immediately
// after commit. The simulated local application stays in a separate process.
func TestExternalKillRecovery(t *testing.T) {
	site := fixture.NewSite()
	bundle := site.Job(1, true, []string{"first", "second"}, "mock-job")
	if path := os.Getenv("ZOP_KILL_DB"); path != "" {
		a := openAgent(t, path, site)
		pause := func() { fmt.Println("READY_TO_KILL"); time.Sleep(30 * time.Second) }
		if os.Getenv("ZOP_KILL_PHASE") == "before commit" {
			a.fault = func(string) error { pause(); return nil }
		}
		if _, err := a.Handle(context.Background(), bundle, sample()); err != nil {
			t.Fatal(err)
		}
		pause()
		return
	}
	clinical := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer clinical.Close()
	for _, phase := range []string{"before commit", "after commit"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "agent.db")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestExternalKillRecovery$")
			cmd.Env = append(os.Environ(), "ZOP_KILL_DB="+path, "ZOP_KILL_PHASE="+phase)
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			scanner := bufio.NewScanner(out)
			ready := scanner.Scan() && scanner.Text() == "READY_TO_KILL"
			err = cmd.Process.Kill()
			waitErr := cmd.Wait()
			if !ready || err != nil || waitErr == nil {
				t.Fatal("kill boundary missed", err, waitErr)
			}
			resp, err := clinical.Client().Get(clinical.URL)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				t.Fatal("simulated product unavailable")
			}
			a := openAgent(t, path, site)
			defer a.Close()
			var pending int
			err = a.db.QueryRow("SELECT count(*) FROM execution_receipt WHERE acknowledged_at IS NULL").Scan(&pending)
			if err != nil || (phase == "before commit" && pending != 0) || (phase == "after commit" && pending != 1) {
				t.Fatal(phase, pending, err)
			}
			if _, err = a.Handle(context.Background(), bundle, sample()); err != nil {
				t.Fatal(err)
			}
			for table, want := range map[string]int{"job_stage": 1, "execution_receipt": 1, "approval_consumption": 2} {
				var n int
				if err = a.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != want {
					t.Fatal(table, n, err)
				}
			}
		})
	}
}
