package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paintersrp/an/internal/config"
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
		inputs:     make([]textinput.Model, 4),
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
			t.Prompt = "Root Vaults Directory: "
			t.Placeholder = fmt.Sprintf("%s/vaults", home)
			t.Focus()
			t.PlaceholderStyle = focusedDimStyle
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
			t.CharLimit = 64
		case 1:
			t.Prompt = "Vault Name: "
			t.Placeholder = "zettel"
			t.PlaceholderStyle = focusedDimStyle
			t.PromptStyle = noStyle
			t.CharLimit = 64
		case 2:
			t.Prompt = "Editor: "
			t.Placeholder = "nvim"
			t.PlaceholderStyle = focusedDimStyle
			t.PromptStyle = noStyle
		case 3:
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

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

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
					RootDir:        m.inputs[0].Value(),
					ActiveVault:    m.inputs[1].Value(),
					Editor:         m.inputs[2].Value(),
					NvimArgs:       m.inputs[3].Value(),
					SubDirs:        []string{defaults[4]},
					FileSystemMode: defaults[5],
					PinnedFile:     defaults[6],
					PinnedTaskFile: defaults[7],
				}

				cfgErr := cfg.Save()
				if cfgErr != nil {
					panic(cfgErr)
				}

				atomsDir := filepath.Join(m.inputs[0].Value(), "atoms")
				err = os.MkdirAll(atomsDir, 0755)
				if err != nil {
					fmt.Printf("Error creating atoms directory: %v\n", err)
				}

				fmt.Println("Initialization complete!")

				return m, tea.Quit
			}

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
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *InitPromptModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

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
		fmt.Sprintf("%s/vaults", path),
		"zettel",
		"nvim",
		"",
		"atoms",
		"confirm",
		"",
		"",
	}
}

func Run() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := config.GetConfigPath(homeDir)

	if _, err := tea.NewProgram(InitialPrompt(path)).Run(); err != nil {
		return err
	}

	return nil
}
