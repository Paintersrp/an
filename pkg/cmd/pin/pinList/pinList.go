package pinList

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/tui/pinList"
)

func NewCmdPinList(s *state.State, pinType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list --print",
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
				return s.Config.ListPins(pinType)
			}

			pinList.Run(s, pinType)
			return nil
		},
	}

	cmd.Flags().
		BoolP("print", "p", false, "Print pins to terminal instead of opening the TUI")

	return cmd
}
