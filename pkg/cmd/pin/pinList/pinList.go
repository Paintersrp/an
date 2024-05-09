package pinList

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func Command(c *config.Config, pinType string) *cobra.Command {

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "List all named pins of a specified type.",
		Long: heredoc.Doc(`
			The pin list command displays all named pins in a structured view.

			Examples:
			  an pin list
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.ListPins(pinType)
		},
	}

	return cmd
}
