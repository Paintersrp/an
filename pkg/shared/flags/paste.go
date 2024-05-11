package flags

import (
	"github.com/spf13/cobra"
)

func AddPaste(cmd *cobra.Command) {
	cmd.Flags().
		Bool("paste", false, "Automatically paste clipboard contents as note content in placeholder.")
}

func HandlePaste(cmd *cobra.Command) (bool, error) {
	return cmd.Flags().GetBool("paste")
}
