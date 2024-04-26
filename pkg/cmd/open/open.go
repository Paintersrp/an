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
package open

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/fzf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdOpen(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open [query]",
		Aliases: []string{"o"},
		Short:   "Open a zettelkasten note.",
		Long: `This command opens a zettelkasten note with nvim, ready for editing.
  It takes one optional argument for a note title, the note to be opened.
  If no title is given, the vault directory files will be displayed
  with a fuzzy finder and file preview for selection, which will pipe into the configured editor to open.`,
		Example: "atomic open cli-notes or atomic open",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			vaultDir := viper.GetString("vaultDir")
			finder := fzf.NewFuzzyFinder(vaultDir)

			if len(args) == 0 {
				finder.Run()
			} else {
				finder.RunWithQuery(args[0])
			}
		},
	}

	return cmd
}
