package state

import (
	"testing"
	"time"

	"github.com/Paintersrp/an/internal/search"
	indexsvc "github.com/Paintersrp/an/internal/services/index"
)

type stubIndexService struct {
	stats indexsvc.Stats
}

func (s stubIndexService) AcquireSnapshot() (*search.Index, error) { return nil, nil }
func (s stubIndexService) QueueUpdate(string)                      {}
func (s stubIndexService) Stats() indexsvc.Stats                   { return s.stats }
func (s stubIndexService) Close() error                            { return nil }

func TestFormatIndexStatusIncludesRebuild(t *testing.T) {
	t.Parallel()

	svc := stubIndexService{stats: indexsvc.Stats{
		Pending:     3,
		LastRebuild: time.Date(2024, time.March, 5, 17, 42, 0, 0, time.UTC),
	}}

	got := formatIndexStatus(svc)
	want := "Idx: pending 3 · rebuilt 17:42"
	if got != want {
		t.Fatalf("formatIndexStatus mismatch: got %q, want %q", got, want)
	}
}

func TestFormatIndexStatusOmitRebuildWhenZero(t *testing.T) {
	t.Parallel()

	svc := stubIndexService{stats: indexsvc.Stats{Pending: 0}}
	got := formatIndexStatus(svc)
	want := "Idx: pending 0"
	if got != want {
		t.Fatalf("formatIndexStatus mismatch: got %q, want %q", got, want)
	}
}

func TestIndexHeartbeatClearsWhenServiceNil(t *testing.T) {
	t.Parallel()

	st := &State{RootStatus: &RootStatus{}}
	st.RootStatus.Set("stale")

	cmd := st.IndexHeartbeatCmd()
	if cmd == nil {
		t.Fatalf("expected heartbeat command")
	}

	msg := cmd()
	statsMsg, ok := msg.(IndexStatsMsg)
	if !ok {
		t.Fatalf("expected IndexStatsMsg, got %T", msg)
	}

	if statsMsg.Line != "" {
		t.Fatalf("expected blank line when index unavailable, got %q", statsMsg.Line)
	}

	if got := st.RootStatus.Value(); got != "" {
		t.Fatalf("expected root status to be cleared, got %q", got)
	}
}

func TestIndexHeartbeatUpdatesStatus(t *testing.T) {
	t.Parallel()

	svc := stubIndexService{stats: indexsvc.Stats{Pending: 7}}
	st := &State{RootStatus: &RootStatus{}, Index: svc}

	cmd := st.IndexHeartbeatCmd()
	if cmd == nil {
		t.Fatalf("expected heartbeat command")
	}

	msg := cmd()
	statsMsg, ok := msg.(IndexStatsMsg)
	if !ok {
		t.Fatalf("expected IndexStatsMsg, got %T", msg)
	}

	want := "Idx: pending 7"
	if statsMsg.Line != want {
		t.Fatalf("expected %q, got %q", want, statsMsg.Line)
	}

	if got := st.RootStatus.Value(); got != want {
		t.Fatalf("expected root status %q, got %q", want, got)
	}
}
