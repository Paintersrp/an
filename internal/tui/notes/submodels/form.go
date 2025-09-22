package submodels

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/utils"
)

type (
	errMsg error
)

const (
	title = iota
	tags
	links
	template
	subdirectory
)

const (
	hotPink  = lipgloss.Color("#0AF")
	darkGray = lipgloss.Color("#767676")
)

var (
	formInputStyle = lipgloss.NewStyle().Foreground(hotPink)
	formTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Background(lipgloss.Color("transparent")).
			Padding(1, 0)

	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

type FormModel struct {
	state                *state.State
	availableTemplates   string
	availableSubdirNames string
	Inputs               []textinput.Model
	availableSubdirs     []string
	Focused              int
	btn                  SubmitButton
}

func NewFormModel(s *state.State) FormModel {
	inputs := make([]textinput.Model, 5)
	inputs[title] = textinput.New()
	inputs[title].Placeholder = "Title"
	inputs[title].Focus()
	inputs[title].CharLimit = 20
	inputs[title].Width = 50
	inputs[title].Prompt = ""

	inputs[tags] = textinput.New()
	inputs[tags].Placeholder = "Tags"
	inputs[tags].CharLimit = 256
	inputs[tags].Width = 50
	inputs[tags].Prompt = ""

	inputs[links] = textinput.New()
	inputs[links].Placeholder = "Links"
	inputs[links].CharLimit = 256
	inputs[links].Width = 50
	inputs[links].Prompt = ""

	inputs[template] = textinput.New()
	inputs[template].Placeholder = "Template"
	inputs[template].CharLimit = 30
	inputs[template].Width = 50
	inputs[template].Prompt = ""

	inputs[subdirectory] = textinput.New()
	inputs[subdirectory].Placeholder = "Subdirectory"
	inputs[subdirectory].CharLimit = 30
	inputs[subdirectory].Width = 50
	inputs[subdirectory].Prompt = ""

	var templateNames []string
	for name := range templater.AvailableTemplates {
		templateNames = append(templateNames, name)
	}

	// Join the template names into a single string separated by commas.
	availableTemplateNames := strings.Join(templateNames, ", ")

	availableSubdirs := s.Handler.GetSubdirectories(s.Vault, "")
	var visibleSubdirs []string
	for _, subdir := range availableSubdirs {
		if !strings.HasPrefix(subdir, ".") {
			visibleSubdirs = append(visibleSubdirs, subdir)
		}
	}
	availableSubdirNames := strings.Join(visibleSubdirs, ", ")

	b := NewSubmitButton()

	return FormModel{
		Inputs:               inputs,
		Focused:              0,
		state:                s,
		availableTemplates:   availableTemplateNames,
		availableSubdirs:     availableSubdirs,
		availableSubdirNames: availableSubdirNames,
		btn:                  b,
	}
}

func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.Inputs)+1) // +1 for the submit button

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.btn.focused {
				return m.handleSubmit(), tea.Quit
			}
			m.nextInput()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN:
			m.nextInput()
		}

		for i := range m.Inputs {
			m.Inputs[i].Blur()
		}
		m.btn.Blur()

		if m.Focused < len(m.Inputs) {
			m.Inputs[m.Focused].Focus()
		} else {
			m.btn.Focus()
		}

		// TODO: Proper error handling
	case errMsg:
		fmt.Println(msg)
		os.Exit(1)
	}

	for i := range m.Inputs {
		m.Inputs[i], cmds[i] = m.Inputs[i].Update(msg)
	}

	var submitCmd tea.Cmd
	m.btn, submitCmd = m.btn.Update(msg)
	cmds[len(cmds)-1] = submitCmd

	return m, tea.Batch(cmds...)
}

func (m FormModel) View() string {
	var btnView string

	if m.btn.focused {
		btnView = formInputStyle.Render(m.btn.View())
	} else {
		btnView = continueStyle.Render(m.btn.View())
	}

	return fmt.Sprintf(
		`
%s
%s
%s

%s
%s

%s
%s

%s
%s

%s
%s

%s
%s

%s
%s

%s
`,
		formTitleStyle.Render("Create a new note"),
		continueStyle.Render("Available Templates:"),
		continueStyle.Width(50).Render(m.availableTemplates),
		continueStyle.Render("Available Subdirectories:"),
		continueStyle.Width(50).Render(m.availableSubdirNames),
		formInputStyle.Width(50).Render("Title"),
		m.Inputs[title].View(),
		formInputStyle.Width(50).Render("Tags (space separated)"),
		m.Inputs[tags].View(),
		formInputStyle.Width(50).Render("Note Links (space separated)"),
		m.Inputs[links].View(),
		formInputStyle.Width(50).Render("Template"),
		m.Inputs[template].View(),
		formInputStyle.Width(50).Render("Subdirectory"),
		m.Inputs[subdirectory].View(),
		btnView,
	) + "\n"
}

func (m *FormModel) nextInput() {
	if m.Focused == len(m.Inputs) {
		if m.btn.focused {
			m.btn.Blur()
			m.Focused = 0
		}

		m.btn.Focus()
	} else {
		m.Focused = (m.Focused + 1) % (len(m.Inputs) + 1) // +1 to include the submit button
	}
}

func (m *FormModel) prevInput() {
	m.Focused--

	if m.Focused == len(m.Inputs) {
		m.btn.Blur()
	}
	if m.Focused < 0 {
		m.Focused = len(m.Inputs) + 1 - 1
	}
}

func (m FormModel) handleSubmit() FormModel {
	title := m.Inputs[title].Value()

	if title == "" {
		return m
	}

	tags, err := utils.ValidateInput(m.Inputs[tags].Value())
	if err != nil {
		return m
	}

	tmpl := m.Inputs[template].Value()

	if _, ok := templater.AvailableTemplates[tmpl]; !ok {
		var templateNames []string
		for name := range templater.AvailableTemplates {
			templateNames = append(templateNames, name)
		}
		availableTemplateNames := strings.Join(templateNames, ", ")

		fmt.Printf(
			"Invalid template specified. Available templates are: %s\n",
			availableTemplateNames,
		)
		return m
	}

	links, err := utils.ValidateInput(m.Inputs[links].Value())
	if err != nil {
		return m
	}

	subDir := m.Inputs[subdirectory].Value()

	if subDir != "" && !m.subdirectoryExists(subDir) {
		fmt.Printf("Subdirectory '%s' does not exist.\n", subDir)
		return m
	}

	n := note.NewZettelkastenNote(
		m.state.Vault,
		subDir,
		title,
		tags,
		links,
		"",
	)

	conflict := n.HandleConflicts()
	if conflict != nil {
		return m
	}

	// TODO: Content instead of "" ?
	note.StaticHandleNoteLaunch(n, m.state.Templater, tmpl, "")

	return m
}

func (m FormModel) subdirectoryExists(subDir string) bool {
	if subDir == "" {
		return true
	}

	for _, dir := range m.availableSubdirs {
		if dir == subDir {
			return true
		}
	}
	return false
}
