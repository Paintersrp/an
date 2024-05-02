package open

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/open/openFind"
	"github.com/Paintersrp/an/pkg/cmd/open/openPin"
	"github.com/Paintersrp/an/pkg/cmd/open/openVault"
	"github.com/spf13/cobra"
)

func NewCmdOpen(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Short:   "Open a zettelkasten note.",
		// Run find by default if only a query / no subcommand
		RunE: openFind.NewCmdOpenFind(c).RunE,
	}

	cmd.AddCommand(openFind.NewCmdOpenFind(c))
	cmd.AddCommand(openVault.NewCmdOpenVault(c))
	cmd.AddCommand(openPin.NewCmdOpenPin(c))

	return cmd
}
