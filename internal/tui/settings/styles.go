package settings

import "github.com/charmbracelet/lipgloss"

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Background(lipgloss.Color("#224")).
			Bold(true).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			Padding(0, 1).Width(100)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0AF")).
				Background(lipgloss.Color("#224")).
				Padding(0, 0)

	focusedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF"))
	cursorStyle = focusedStyle.Copy()
	textStyle   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCC"))
)
