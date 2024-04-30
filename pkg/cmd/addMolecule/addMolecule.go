package addMolecule

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdAddMolecule(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-molecule [name]",
		Aliases: []string{"am", "add-m", "a-m"},
		Short:   "Add a molecule to the list of available projects.",
		Long:    `This command adds a molecule to the list of available projects in the global persistent configuration.`,
		Example: "atomic add-molecule Go-CLI",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			c.AddMolecule(name)
		},
	}

	return cmd
}
