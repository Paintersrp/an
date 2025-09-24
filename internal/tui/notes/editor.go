package notes

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/tui/textarea"
)

type editorMode int

const (
	editorModeNone editorMode = iota
	editorModeExisting
	editorModeScratch
)

type editorSession struct {
	area             *textarea.Model
	mode             editorMode
	path             string
	title            string
	originalContent  string
	originalChecksum [32]byte
	originalModTime  time.Time
	pendingDiscard   bool
	allowOverwrite   bool
	status           string
}

func newEditorSession(width, height int) *editorSession {
	return &editorSession{area: textarea.New(width, height)}
}

func (s *editorSession) setMetadata(path, title string, mode editorMode) {
	s.path = path
	s.title = title
	s.mode = mode
}

func (s *editorSession) setOriginal(content string, modTime time.Time) {
	s.originalContent = content
	s.originalChecksum = sha256.Sum256([]byte(content))
	s.originalModTime = modTime
	s.allowOverwrite = false
	s.pendingDiscard = false
}

func (s *editorSession) hasChanges() bool {
	if s == nil || s.area == nil {
		return false
	}
	return s.area.Value() != s.originalContent
}

func (s *editorSession) viewHeader() string {
	if s == nil {
		return ""
	}
	switch s.mode {
	case editorModeExisting:
		return fmt.Sprintf("Editing %s", filepath.Base(s.title))
	case editorModeScratch:
		return "Scratch capture"
	default:
		return ""
	}
}

func (s *editorSession) setSize(width, height int) {
	if s == nil || s.area == nil {
		return
	}
	s.area.SetSize(width, height)
}

func (s *editorSession) focus() tea.Cmd {
	if s == nil || s.area == nil {
		return nil
	}
	return s.area.Focus()
}

func (s *editorSession) blur() tea.Cmd {
	if s == nil || s.area == nil {
		return nil
	}
	return s.area.Blur()
}

func (s *editorSession) setValue(content string) {
	if s == nil || s.area == nil {
		return
	}
	s.area.SetValue(content)
	s.pendingDiscard = false
}

func (s *editorSession) value() string {
	if s == nil || s.area == nil {
		return ""
	}
	return s.area.Value()
}

func (s *editorSession) checksumMatches(content []byte) bool {
	if s == nil {
		return true
	}
	sum := sha256.Sum256(content)
	return sum == s.originalChecksum
}
