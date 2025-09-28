package state

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// IndexStatsMsg notifies subscribers that the root status line was refreshed
// using the latest search index statistics.
type IndexStatsMsg struct {
	Line string
}

// IndexHeartbeatCmd polls the index service for lightweight statistics,
// updates the shared root status line, and returns a message that consumers can
// use to trigger rerenders.
func (s *State) IndexHeartbeatCmd() tea.Cmd {
	if s == nil {
		return nil
	}

	return func() tea.Msg {
		line := formatIndexStatus(s.Index)
		if s.RootStatus != nil {
			s.RootStatus.Set(line)
		}
		return IndexStatsMsg{Line: line}
	}
}

func formatIndexStatus(svc IndexService) string {
	if svc == nil {
		return ""
	}

	stats := svc.Stats()
	parts := []string{fmt.Sprintf("Idx: pending %d", stats.Pending)}
	if !stats.LastRebuild.IsZero() {
		parts = append(parts, fmt.Sprintf("rebuilt %s", formatRebuildTime(stats.LastRebuild)))
	}

	return strings.Join(parts, " · ")
}

func formatRebuildTime(t time.Time) string {
	return t.Local().Format("15:04")
}
