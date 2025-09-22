package submodels

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF"))

	cursorStyle = focusedStyle.Copy()
)

type InputModel struct {
	Title      string
	Input      textinput.Model
	cursorMode cursor.Mode
}

func NewInputModel() InputModel {
	t := textinput.New()
	t.Cursor.Style = cursorStyle
	t.PromptStyle = focusedStyle
	t.TextStyle = focusedStyle
	t.Focus()

	m := InputModel{
		Title: "",
		Input: t,
	}

	return m
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}

			return m, tea.Batch(m.Input.Cursor.SetMode(m.cursorMode))
		}
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)

	return m, cmd
}

func (m InputModel) View() string {
	var b strings.Builder
	b.WriteString(m.Input.View())
	return b.String()
}
