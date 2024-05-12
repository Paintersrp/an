package initialize

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
)

var (
	focusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cba6f7"))
	focusedDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#585b70"))
	blurredStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cba6f7"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle.Copy()
	cursorModeHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cba6f7"))

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf(
		"[ %s ]",
		blurredStyle.Render("Submit"),
	)
)

type InitPromptModel struct {
	configPath string
	inputs     []textinput.Model
	focusIndex int
	cursorMode cursor.Mode
}

func InitialPrompt(cfgPath string) InitPromptModel {
	m := InitPromptModel{
		inputs:     make([]textinput.Model, 3),
		configPath: cfgPath,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch i {
		case 0:
			t.Prompt = "Notes (Vault) Directory: "
			t.Placeholder = fmt.Sprintf("%s/notes", home)
			t.Focus()
			t.PlaceholderStyle = focusedDimStyle
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
			t.CharLimit = 64
		case 1:
			t.Prompt = "Editor: "
			t.Placeholder = "nvim"
			t.PlaceholderStyle = focusedDimStyle
			t.PromptStyle = noStyle
		case 2:
			t.Prompt = "Editor Arguments: "
			t.Placeholder = "none"
			t.PlaceholderStyle = focusedDimStyle
			t.PromptStyle = noStyle
		}

		m.inputs[i] = t
	}

	return m
}

func (m InitPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InitPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				cmds[i] = m.inputs[i].Cursor.SetMode(m.cursorMode)
			}
			return m, tea.Batch(cmds...)

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, execute and exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {

				home, err := os.UserHomeDir()
				if err != nil {
					panic(err)
				}

				defaults := SetupDefaults(home)

				for i := range m.inputs {
					if i != len(m.inputs) {
						if m.inputs[i].Value() == "" {
							m.inputs[i].SetValue(defaults[i])
						}
					}
				}

				cfg := &config.Config{
					VaultDir:       m.inputs[0].Value(),
					Editor:         m.inputs[1].Value(),
					NvimArgs:       m.inputs[2].Value(),
					SubDirs:        []string{defaults[3]},
					FileSystemMode: defaults[4],
					PinnedFile:     defaults[5],
					PinnedTaskFile: defaults[6],
				}

				cfgErr := cfg.Save()
				if cfgErr != nil {
					panic(cfgErr)
				}
				fmt.Println("Initialization complete!")

				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *InitPromptModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m InitPromptModel) View() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyle.Render("cursor mode is "))
	b.WriteString(cursorModeHelpStyle.Render(m.cursorMode.String()))
	b.WriteString(helpStyle.Render(" (ctrl+r to change style)"))
	b.WriteString(
		helpStyle.Render("\n(Leave inputs blank for default values)"),
	)

	return b.String()
}

func SetupDefaults(path string) []string {
	return []string{
		fmt.Sprintf("%s/notes", path),
		"nvim",
		"",
		"atoms",
		"confirm",
		"",
		"",
	}
}

func Run(s *state.State) error {
	if _, err := tea.NewProgram(InitialPrompt(s.Config.GetConfigPath())).Run(); err != nil {
		return err
	}

	return nil
}
