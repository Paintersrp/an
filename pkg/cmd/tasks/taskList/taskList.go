package taskList

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/fs/parser"
	"github.com/Paintersrp/an/internal/config"
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
