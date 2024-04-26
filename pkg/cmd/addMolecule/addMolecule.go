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
package addMolecule

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdAddMolecule(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-molecule [name]",
		Aliases: []string{"am", "add-m", "a-m"},
		Short:   "Add a molecule to the list of available projects.",
		Long:    `This command adds a molecule to the list of available projects in the global persistent configuration.`,
		Example: "atomic add-molecule Go-CLI",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			c.AddMolecule(name)
		},
	}

	return cmd
}
