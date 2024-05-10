package pinRemove

import (
	"errors"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
)

func Command(c *config.Config, pinType string) *cobra.Command {
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

			return c.DeleteNamedPin(name, pinType, true)
		},
	}

	cmd.Flags().
		StringVarP(&name, "name", "n", "", "The name of the pin to unpin (required)")
	cmd.MarkFlagRequired("name")

	return cmd
}
