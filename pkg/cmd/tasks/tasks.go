/*
Copyright © 2024 Ryan Painter paintersrp@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package tasks

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskEcho"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskList"
	"github.com/Paintersrp/an/pkg/cmd/tasks/taskPin"
	"github.com/spf13/cobra"
)

func NewCmdTasks(c *config.Config) *cobra.Command {
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

	return cmd
}
