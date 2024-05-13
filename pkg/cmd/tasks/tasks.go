package tasks

import (
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskEcho"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskList"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskNewEchoFile"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskOpenPin"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskPin"
)

func NewCmdTasks(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Task management and operations",
		Long:  `The tasks command is used for managing and operating on tasks within the system.`,
		Example: `
    # List all tasks
    an-cli tasks list

    # Pin a task file
    an-cli tasks pin ~/tasks.md

    # Echo a new task into the pinned task file
    an-cli tasks echo "Finish the report" -p high
    `,
	}

	cmd.AddCommand(taskEcho.NewCmdTaskEcho(s))
	cmd.AddCommand(taskList.NewCmdTasksList(s))
	cmd.AddCommand(taskPin.NewCmdTaskPin(s))
	cmd.AddCommand(taskNewEchoFile.NewCmdNewEchoFile(s))
	cmd.AddCommand(taskOpenPin.NewCmdTaskOpenPin(s))

	return cmd
}
