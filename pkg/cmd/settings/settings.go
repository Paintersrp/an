package settings

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/tui/list"
	"github.com/spf13/cobra"
)

func NewCmdSettings(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Aliases: []string{"s"},
		Short:   "CLI settings menu",
		Long:    "This command allows you to adjusts your settings directly from the CLI tool.",
		Example: "an settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := list.Run(c); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
