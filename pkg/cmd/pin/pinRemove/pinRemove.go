package pinRemove

import (
	"errors"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/spf13/cobra"
)

func Command(s *state.State, pinType string) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "remove [--name pin_name]",
		Aliases: []string{"r"},
		Short:   "Unpin a named file or task.",
		Long: heredoc.Doc(`
			The unpin command removes a named pin or task pin from the configuration.
			You need to specify the name of the pin.

			Examples:
			  an pin remove --name my-note
			  an pin remove --name my-task
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return errors.New("you must specify a name for the pin to unpin")
			}

			return s.Config.DeleteNamedPin(name, pinType, true)
		},
	}

	cmd.Flags().
		StringVarP(&name, "name", "n", "", "The name of the pin to unpin (required)")
	cmd.MarkFlagRequired("name")

	return cmd
}
