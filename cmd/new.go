/*
Copyright © 2024 Ryan Painter <paintersrp@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Paintersrp/zet-cli/internal/zet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var newCmd = &cobra.Command{
	Use:   "new [title] [tags]",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("error: No title given. Try again with zet-cli new [title]")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]
		var tags []string

		if len(args) > 1 {
			tags = strings.Split(args[1], " ")

			for i, s := range tags {
				fmt.Printf("Tag #%s: %s\n", fmt.Sprint(i+1), s)
			}
		}

		vaultDir := viper.GetString("vaultdir")
		note := zet.NewZettelkastenNote(vaultDir, title, tags)

		exists, _, existsErr := note.FileExists()
		if existsErr != nil {
			return existsErr
		}

		if exists {
			fmt.Println("error: Note with given title already exists in the vault directory.")
			fmt.Println("hint: Try again with a new title, or run again with either an overwrite (-o) flag or an increment (-i) flag")
			os.Exit(1)
		}

		_, createErr := note.Create()
		if createErr != nil {
			return createErr
		}

		// Open the note in Neovim.
		if err := note.Open(); err != nil {
			fmt.Println("Error opening note in Neovim:", err)
			os.Exit(1)
		}

		return nil
	},
}

// var newCmd = &cobra.Command{
// 	Use:   "new [title] [tags]",
// 	Short: "A brief description of your command",
// 	Long: `A longer description that spans multiple lines and likely contains examples
// and usage of using your command.`,
// 	Args: func(cmd *cobra.Command, args []string) error {
// 		if len(args) == 0 {
// 			return fmt.Errorf("error: No title given. Try again with zet-cli new [title]")
// 		}
// 		return nil
// 	},
// 	Run: func(cmd *cobra.Command, args []string) {
// 		title := args[0]
// 		var tags []string
//
// 		if len(args) > 1 {
// 			tags = strings.Split(args[1], " ")
//
// 			for i, s := range tags {
// 				fmt.Printf("Tag #%s: %s\n", fmt.Sprint(i+1), s)
// 			}
// 		}
//
// 		vaultDir := viper.GetString("vaultdir")
//
// 		noteFilePath := filepath.Join(vaultDir, title+".md")
// 		if _, err := os.Stat(noteFilePath); err == nil {
// 			fmt.Println("error: Note with given title already exists in the vault directory.")
// 			fmt.Println("hint: Try again with a new title, or run again with either an overwrite (-o) flag or an increment (-i) flag")
// 			os.Exit(1)
// 		}
//
// 		// Create a new zettelkasten note.
// 		note, err := zet.NewZettelkastenNote(vaultDir, title, tags)
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}
//
// 		// Open the note in Neovim.
// 		if err := note.Open(); err != nil {
// 			fmt.Println("Error opening note in Neovim:", err)
// 			os.Exit(1)
// 		}
// 	},
// }

// TODO -o and -i flags
func init() {
	rootCmd.AddCommand(newCmd)

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// newCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
