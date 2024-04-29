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
package initialize

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/tui/initialize"
	"github.com/spf13/cobra"
)

func NewCmdInit(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "initialize",
		Aliases: []string{"i", "init"},
		Short:   "initialize zet-cli",
		Long:    "This command will walk you through setting up and initializing your zet-cli's configuration.",
		Example: "zet init",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initialize.Run(c); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
