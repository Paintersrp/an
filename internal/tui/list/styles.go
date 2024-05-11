package list

import "github.com/charmbracelet/lipgloss"

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("36")).
			BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1).Width(80)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render

	focusedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	cursorStyle  = focusedStyle.Copy()
	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#EE6FF8"))
)
