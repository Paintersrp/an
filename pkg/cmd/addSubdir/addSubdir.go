package addSubdir

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdAddSubdir(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-subdir [name]",
		Aliases: []string{"as", "add-s", "a-s"},
		Short:   "Add a sub directory to the list of available directories.",
		Long:    `This command adds a sub directory to the list of available projects in the global persistent configuration.`,
		Example: "atomic add-subdir Go-CLI",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			c.AddSubdir(name)
		},
	}

	return cmd
}
