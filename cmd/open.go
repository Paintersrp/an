/*
Copyright © 2024 Ryan Painter <paintersrp@gmail.com>
*/
package cmd

import (
	"github.com/Paintersrp/an/internal/fzf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// openCmd represents the open command
var openCmd = &cobra.Command{
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

func init() {
	rootCmd.AddCommand(openCmd)
}
