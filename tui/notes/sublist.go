// Unused sublist model, intended to be an alternative view for linking an orphan note
// Overall I did not like the flow of the system, nor did I find it all that useful especially
// when factoring in all the caveats and concerns that would need to be handled for adjusting the note upstream
// in this manner. Still, I may refine it later
package notes

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type SubListModel struct {
	views             map[string]ViewConfig
	config            *config.Config
	list              list.Model
	width             int
	height            int
	showAsFileDetails bool
}

func NewSubListModel(
	cfg *config.Config,
	views map[string]ViewConfig,
) SubListModel {
	files, _ := GetFilesByView(views, "default", cfg.VaultDir)
	items := ParseNoteFiles(files, cfg.VaultDir, false)

	// Setup list
	d := list.NewDefaultDelegate()
	l := list.New(items, d, 0, 0)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(false)
	l.Title = "Select a Note to Link (Upstream)\nUse Esc or Q to Close the Link Selection View"
	l.Styles.Title = titleStyle

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "exit link view"),
			)}
	}

	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "exit link view"),
			)}
	}

	return SubListModel{
		list:   l,
		config: cfg,
		views:  views,
	}
}

func (m SubListModel) Init() tea.Cmd {
	return nil
}

func (m SubListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
	default:
		return m, nil
	}

	nl, cmd := m.list.Update(msg)
	m.list = nl
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m SubListModel) View() string {
	return m.list.View()
}
