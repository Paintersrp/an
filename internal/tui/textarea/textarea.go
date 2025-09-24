package textarea

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is a lightweight wrapper around Bubble's textarea model that exposes
// a focused text editor surface for embedding in other TUI components.
type Model struct {
	textarea textarea.Model
}

// New creates a textarea model sized to the provided dimensions. The editor is
// configured without a character limit and with a blank prompt so callers can
// embed it inside their own layouts without extra framing.
func New(width, height int) *Model {
	ti := textarea.New()
	ti.Placeholder = ""
	ti.Prompt = ""
	ti.CharLimit = 0
	if width > 0 {
		ti.SetWidth(width)
	}
	if height > 0 {
		ti.SetHeight(height)
	}

	return &Model{textarea: ti}
}

// Init satisfies the Bubble Tea Model interface so callers can batch the blink
// command when integrating the textarea into larger update loops.
func (m *Model) Init() tea.Cmd {
	if m == nil {
		return nil
	}
	return textarea.Blink
}

// Update applies the incoming message to the underlying textarea and returns
// the resulting command. Callers are responsible for handling higher-level
// keyboard shortcuts such as save/discard.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m == nil {
		return nil
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return cmd
}

// View renders the textarea contents.
func (m *Model) View() string {
	if m == nil {
		return ""
	}
	return m.textarea.View()
}

// Focus applies focus to the textarea returning the generated command.
func (m *Model) Focus() tea.Cmd {
	if m == nil {
		return nil
	}
	return m.textarea.Focus()
}

// Blur removes focus from the textarea.
func (m *Model) Blur() tea.Cmd {
	if m == nil {
		return nil
	}
	m.textarea.Blur()
	return nil
}

// SetValue replaces the textarea contents with the provided value while
// preserving the cursor at the end of the buffer.
func (m *Model) SetValue(value string) {
	if m == nil {
		return
	}
	m.textarea.SetValue(value)
	m.textarea.CursorEnd()
}

// Value returns the current textarea contents.
func (m *Model) Value() string {
	if m == nil {
		return ""
	}
	return m.textarea.Value()
}

// CursorEnd positions the cursor at the end of the textarea's buffer.
func (m *Model) CursorEnd() {
	if m == nil {
		return
	}
	m.textarea.CursorEnd()
}

// SetSize updates the textarea dimensions.
func (m *Model) SetSize(width, height int) {
	if m == nil {
		return
	}
	if width > 0 {
		m.textarea.SetWidth(width)
	}
	if height > 0 {
		m.textarea.SetHeight(height)
	}
}
