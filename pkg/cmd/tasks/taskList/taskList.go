package taskList

import (
	"fmt"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/parser"
	"github.com/spf13/cobra"
)

func NewCmdTasksList(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "List all tasks",
		Long:    `The list command displays all the tasks in a tabular format.`,
		Example: `
    # List all tasks
    an-cli tasks list
    `,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(c)
		},
	}

	return cmd
}

func run(c *config.Config) error {
	p := parser.NewParser(c.VaultDir)

	if err := p.Walk(); err != nil {
		fmt.Println("Error:", err)
		return err
	}

	p.ShowTasksTable()

	return nil
}
