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
