/*
Copyright © 2024 Ryan Painter <paintersrp@gmail.com>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var changeEditorCmd = &cobra.Command{
	Use:   "change-editor [editor]",
	Short: "",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appCfg.ChangeEditor(args[0])
	},
}

func init() {
	rootCmd.AddCommand(changeEditorCmd)
}
