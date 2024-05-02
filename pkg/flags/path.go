package flags

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func AddPath(cmd *cobra.Command) {
	cmd.Flags().
		StringP(
			"path",
			"p",
			"",
			"Manually enter the path to the file to pin without fuzzyfinding",
		)
}

func HandlePath(cmd *cobra.Command) string {
	path, err := cmd.Flags().GetString("path")
	if err != nil {
		fmt.Printf("error retrieving path flag: %s\n", err)
		os.Exit(1)
	}
	return path
}
