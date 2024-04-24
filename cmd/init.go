package cmd

import (
	"github.com/Paintersrp/zet-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:     "initialize",
	Short:   "initialize zet-cli",
	Long:    "This command will walk you through setting up and initializing your zet-cli's configuration.",
	Example: "zet init",
	Aliases: []string{"i", "init"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := tea.NewProgram(tui.InitialPrompt(viper.ConfigFileUsed())).Run(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
