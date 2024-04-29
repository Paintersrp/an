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
package tags

import (
	"fmt"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/parser"
	"github.com/spf13/cobra"
)

func NewCmdTags(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "",
		Long:  ``,
		// Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			p := parser.NewParser(c.VaultDir)

			if err := p.Walk(); err != nil {
				fmt.Println("Error:", err)
				return
			}

			p.ShowTagTable()
		},
	}

	return cmd
}
