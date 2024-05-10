package pinList

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/tui/pinList"
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
			print, err := cmd.Flags().GetBool("print")
			if err != nil {
				return err
			}

			if print {
				return c.ListPins(pinType)
			}

			pinList.Run(c, pinType)
			return nil
		},
	}

	cmd.Flags().
		BoolP("print", "p", false, "Print pins to terminal instead of opening the TUI")

	return cmd
}
