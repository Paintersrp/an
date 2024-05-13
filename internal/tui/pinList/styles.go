package pinList

import "github.com/charmbracelet/lipgloss"

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Background(lipgloss.Color("#224")).
			Bold(true).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#0AF", Dark: "#0AF"}).
				Render

	listStyle = lipgloss.NewStyle().
			MarginRight(1).
			Border(lipgloss.NormalBorder(), false, false, false, false).
			BorderForeground(lipgloss.Color("#334455"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0AF")).
				Background(lipgloss.Color("#224")).
				Padding(0, 0)
)
