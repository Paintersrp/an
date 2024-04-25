/*
Copyright © 2024 Ryan Painter <paintersrp@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/internal/zet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var newCmd = &cobra.Command{
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

		_, createErr := note.Create(tmpl, appTemplater)
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

// TODO -overwrite and -increment flags ?
func init() {
	newCmd.Flags().
		StringP("template", "t", "zet", "Specify the template to use (default is 'zet'). Available templates: daily, roadmap, zet")
	viper.BindPFlag(
		"template",
		newCmd.Flags().Lookup("template"),
	)
	rootCmd.AddCommand(newCmd)
}
