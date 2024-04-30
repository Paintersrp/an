package pin

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin [file path]",
		Short: "Pin a file to be used with the echo command.",
		Long: `The pin command allows the user to specify a file that can be used with the echo command.
The path to the pinned file is saved in the configuration.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			c.ChangePin(filePath, "text")
			return nil
		},
	}
	return cmd
}
