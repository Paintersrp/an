package flags

import (
	"github.com/spf13/cobra"
)

func AddName(cmd *cobra.Command, usage string) {
	cmd.Flags().StringP("name", "n", "", usage)
}

func HandleName(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("template")
}
