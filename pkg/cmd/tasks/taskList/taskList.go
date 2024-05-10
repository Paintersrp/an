package taskList

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/fs/parser"
	"github.com/Paintersrp/an/internal/state"
)

func NewCmdTasksList(s *state.State) *cobra.Command {
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
			return run(s)
		},
	}

	return cmd
}

func run(s *state.State) error {
	p := parser.NewParser(s.Vault)

	if err := p.Walk(); err != nil {
		fmt.Println("Error:", err)
		return err
	}

	p.ShowTasksTable()

	return nil
}
