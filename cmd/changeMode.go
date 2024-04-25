/*
Copyright © 2024 Ryan Painter <paintersrp@gmail.com>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

// changeModeCmd represents the changeMode command
var changeModeCmd = &cobra.Command{
	Use:   "change-mode [mode]",
	Short: "",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appCfg.ChangeMode(args[0])
	},
}

func init() {
	rootCmd.AddCommand(changeModeCmd)
}
