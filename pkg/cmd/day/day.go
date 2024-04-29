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
package day

import (
	"fmt"
	"time"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdDay(
	c *config.Config,
	t *templater.Templater,
) *cobra.Command {
	var index int
	cmd := &cobra.Command{
		Use:     "day",
		Aliases: []string{"d"},
		Short:   "Create or open a daily note.",
		Long: `
  This command creates or opens a daily note based on the given index.
  The index can be negative for past days, positive for future days, or zero for today.

  Examples:
  an-cli day --index -1  // Opens yesterday's note
  an-cli day --index +1  // Creates or opens tomorrow's note
  an-cli day             // Opens today's note (default index is 0)
  `,
		RunE: func(cmd *cobra.Command, args []string) error {
			date := time.Now().AddDate(0, 0, index).Format("20060102")
			tmpl := "day" // Default template for daily notes

			vaultDir := viper.GetString("vaultdir")
			note := zet.NewZettelkastenNote(
				vaultDir,
				"atoms", // default to only atoms for day files, can change later if want to
				fmt.Sprintf(
					"day-%s",
					date,
				), // Empty title - Day template autotitles with date
				[]string{},
			)

			exists, _, existsErr := note.FileExists()
			if existsErr != nil {
				return existsErr
			}

			if exists {
				return note.Open()
			}

			_, createErr := note.Create(tmpl, t)
			if createErr != nil {
				return createErr
			}

			return note.Open()
		},
	}

	cmd.Flags().
		IntVarP(&index, "index", "i", 0, "Index for the day relative to today. Can be negative for past days or positive for future days.")
	return cmd
}
