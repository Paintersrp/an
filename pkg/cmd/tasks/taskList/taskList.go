package taskList

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	services "github.com/Paintersrp/an/internal/services/tasks"
	"github.com/Paintersrp/an/internal/state"
	tableTui "github.com/Paintersrp/an/internal/tui/table"
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
	if s == nil || s.Handler == nil || s.Tasks == nil {
		return fmt.Errorf("task list requires a configured state handler")
	}

	svc := services.NewService(s.Handler, s.Tasks)
	items, err := svc.List()
	if err != nil {
		return err
	}

	tableModel := services.TableFromItems(items, 20)
	program := tea.NewProgram(tableTui.NewTableModel(tableModel))
	if _, err := program.Run(); err != nil {
		return err
	}

	return nil
}
