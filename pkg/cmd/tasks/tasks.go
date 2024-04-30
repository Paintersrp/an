package tasks

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskEcho"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskList"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskNewEchoFile"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskOpenPin"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskPin"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/spf13/cobra"
)

func NewCmdTasks(c *config.Config, t *templater.Templater) *cobra.Command {
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

	cmd.AddCommand(taskEcho.NewCmdTaskEcho(c))
	cmd.AddCommand(taskPin.NewCmdTaskPin(c))
	cmd.AddCommand(taskList.NewCmdTasksList(c))
	cmd.AddCommand(taskNewEchoFile.NewCmdNewEchoFile(c, t))
	cmd.AddCommand(taskOpenPin.NewCmdTaskOpenPin(c))

	return cmd
}
