package views

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	viewsadd "github.com/Paintersrp/an/pkg/cmd/views/add"
	viewsremove "github.com/Paintersrp/an/pkg/cmd/views/remove"
)

func NewCmdViews(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "views",
		Short: "Manage custom views",
		Long: heredoc.Doc(`
                        Manage your custom note views. Use the subcommands to add or remove views without editing
                        the configuration file manually.
                `),
	}

	cmd.AddCommand(
		viewsadd.NewCmdViewAdd(s),
		viewsremove.NewCmdViewRemove(s),
	)

	return cmd
}
