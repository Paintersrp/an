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
package new

import (
	"fmt"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdNew(
	c *config.Config,
	t *templater.Templater,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [title] [tags]",
		Aliases: []string{"n"},
		Short:   "Create a new zettelkasten note.",
		Long: `
  This command creates a new atomic kettelkasten note into your note vault directory.
  It takes a required title argument and an optional tags argument to quickly add tags to the newly made note.

              [title]  [tags]
  zet-cli new robotics "robotics science class study-notes"
  `,
		Example: "atomic new cli-notes 'cli go zettelkasten notetaking learn'",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf(
					"error: No title given. Try again with zet-cli new [title]",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			var tags []string

			if len(args) > 1 {
				tags = strings.Split(args[1], " ")
			}

			tmpl := viper.GetString("template")
			if _, ok := templater.AvailableTemplates[tmpl]; !ok {
				return fmt.Errorf(
					"error: Invalid template specified. Available templates are: daily, roadmap, zet",
				)
			}

			moleculeFlag := viper.GetString("molecule")

			vaultDir := viper.GetString("vaultdir")
			note := zet.NewZettelkastenNote(
				vaultDir,
				moleculeFlag,
				title,
				tags,
			)

			exists, _, existsErr := note.FileExists()
			if existsErr != nil {
				return existsErr
			}

			if exists {
				fmt.Println(
					"error: Note with given title already exists in the vault directory.",
				)
				fmt.Println(
					"hint: Try again with a new title, or run again with either an overwrite (-o) flag or an increment (-i) flag",
				)
				os.Exit(1)
			}

			_, createErr := note.Create(tmpl, t)
			if createErr != nil {
				return createErr
			}

			// Open the note in Neovim.
			if err := note.Open(); err != nil {
				fmt.Println(
					"Error opening note in Neovim:",
					err,
				)
				os.Exit(1)
			}

			return nil
		},
	}
	cmd.Flags().
		StringP("template", "t", "zet", "Specify the template to use (default is 'zet'). Available templates: daily, roadmap, zet")
	viper.BindPFlag(
		"template",
		cmd.Flags().Lookup("template"),
	)

	return cmd
}
