package fleeting

import (
	"fmt"
	"log"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func NewCmdFleeting(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fleeting",
		Aliases: []string{"fleet", "f"},
		Short:   "",
		Long:    heredoc.Doc(``),
		Example: heredoc.Doc(`
			an fleeting
			an fleet
			an f
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(initialModel())

			if _, err := p.Run(); err != nil {
				log.Fatal(err)
			}

			return nil
		},
	}

	return cmd
}

type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel() model {
	ti := textarea.New()
	ti.Placeholder = "..."
	ti.Focus()

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
