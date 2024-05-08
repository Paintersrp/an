package textarea

import (
	"fmt"
	"log"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel(yank bool) model {
	ti := textarea.New()
	ti.Placeholder = "..."
	ti.SetHeight(40)
	ti.CharLimit = 0
	ti.SetWidth(100)
	ti.Focus()

	if yank {
		// Get the clipboard content
		content, err := clipboard.ReadAll()
		if err != nil {
			log.Fatal(err)
		}
		// Set the clipboard content as the initial text in the textarea
		ti.SetValue(content)
	}

	return model{
		textarea: ti,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlS:
			fmt.Println("Saving note...")
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return fmt.Sprintf(
		"Write note content.\n\n%s\n\n%s\n%s",
		m.textarea.View(),
		"(ctrl+c to quit)",
		"(ctrl+s to save note)",
	) + "\n\n"
}
