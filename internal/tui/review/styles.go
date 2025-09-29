package review

import "github.com/charmbracelet/lipgloss"

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Background(lipgloss.Color("transparent")).
			Bold(true).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#0AF", Dark: "#0AF"})

	tabActiveStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Underline(true)

	tabInactiveStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Faint(true)

	historyPreviewStyle = lipgloss.NewStyle().
				MarginLeft(4)
)
