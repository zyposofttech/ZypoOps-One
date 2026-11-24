package agent

import (
	"context"
	"path/filepath"
	"testing"
	"zyposoft.local/operations/packages/signing-go"
	"zyposoft.local/operations/tests/fixtures"
)

func TestDistinctJobCannotReuseIdempotencyKey(t *testing.T) {
	s := fixture.NewSite()
	a := openAgent(t, filepath.Join(t.TempDir(), "agent.db"), s)
	defer a.Close()
	initial := s.Job(1, false, nil, "mock-job")
	if _, err := a.Handle(context.Background(), initial, sample()); err != nil {
		t.Fatal(err)
	}
	second := s.Job(2, false, nil, "rotated-job")
	doc := fixture.Doc(second.Job)
	doc["body"].(signing.Object)["idempotency_key"] = fixture.Doc(initial.Job)["body"].(signing.Object)["idempotency_key"]
	second.Job = s.Sign(doc, "rotated-job")
	if _, err := a.Handle(context.Background(), second, sample()); err == nil || err.Error() != "IDEMPOTENCY_CONFLICT" {
		t.Fatal(err)
	}
	if _, err := a.Handle(context.Background(), s.Job(2, false, nil, "rotated-job"), sample()); err != nil {
		t.Fatal("rejection advanced sequence", err)
	}
}
