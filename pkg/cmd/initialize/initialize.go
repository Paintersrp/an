package initialize

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/tui/initialize"
	"github.com/spf13/cobra"
)

func NewCmdInit(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "initialize",
		Aliases: []string{"i", "init"},
		Short:   "initialize zet-cli",
		Long:    "This command will walk you through setting up and initializing your zet-cli's configuration.",
		Example: "zet init",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initialize.Run(c); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
