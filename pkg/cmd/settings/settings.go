package settings

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/tui/settings"
)

func NewCmdSettings(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Aliases: []string{"s"},
		Short:   "CLI settings menu",
		Long: heredoc.Doc(`
			This command opens the settings menu, allowing you to adjust your CLI tool's settings.
			You can customize your application behavior through various options, ensuring that the
			CLI adapts to your workflow and preferences.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := settings.Run(c); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
