package taskPin

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdTaskPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pin [file path]",
		Aliases: []string{"p"},
		Short:   "Pin a task file",
		Long:    `The pin command is used to pin a task file, making it the target for other task operations.`,
		Example: `
    # Pin a task file
    an-cli tasks pin ~/tasks.md
    `,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			c.ChangePin(filePath, "task")
			return nil
		},
	}
	return cmd
}
