package notes

import (
	"fmt"

	"github.com/Paintersrp/an/internal/config"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type SubListModel struct {
	modes             map[string]ModeConfig
	config            *config.Config
	List              list.Model
	width             int
	height            int
	showAsFileDetails bool
}

func NewSubListModel(
	cfg *config.Config,
	modes map[string]ModeConfig,
) SubListModel {
	files, _ := getFilesByMode(modes, "default", cfg.VaultDir)
	items := parseNoteFiles(files, cfg.VaultDir, false)
	fmt.Println(items)

	// Setup list
	d := list.NewDefaultDelegate()
	l := list.New(items, d, 0, 0)
	l.Title = "Picker List"
	l.Styles.Title = titleStyle

	return SubListModel{
		List:   l,
		config: cfg,
		modes:  modes,
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
		h, v := appStyle.GetFrameSize()
		m.List.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
	default:
		return m, nil
	}

	nl, cmd := m.List.Update(msg)
	m.List = nl
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m SubListModel) View() string {
	return appStyle.Render(m.List.View())
}
