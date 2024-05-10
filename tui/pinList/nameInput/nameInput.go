package nameInput

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#0AF"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle.Copy()

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Background(lipgloss.Color("#224")).
			Bold(true).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Bold(true).
			Padding(0, 1)
)

type NameInputModel struct {
	Title      string
	Input      textinput.Model
	cursorMode cursor.Mode
}

func NewNameInput() NameInputModel {
	t := textinput.New()
	t.Cursor.Style = cursorStyle
	t.PromptStyle = focusedStyle
	t.TextStyle = focusedStyle

	t.Focus()

	m := NameInputModel{
		Title: "",
		Input: t,
	}

	return m
}

func (m NameInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m NameInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Change cursor mode
		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}

			return m, tea.Batch(m.Input.Cursor.SetMode(m.cursorMode))
		}
	}

	// Handle character input and blinking
	_, cmd := m.Update(msg)

	return m, cmd
}

func (m NameInputModel) View() string {
	var b strings.Builder
	title := titleStyle.Render("Enter new name for pin") + "\n\n"
	help := "\n\n" + helpStyle.Render(
		"(enter to submit)",
	) + "\n" + helpStyle.Render(
		"(esc/q to exit)",
	)

	b.WriteString(title)
	b.WriteString(m.Input.View())
	b.WriteString(help)
	return b.String()
}
