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
