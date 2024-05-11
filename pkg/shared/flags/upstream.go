package flags

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/fzf"
)

func AddUpstream(cmd *cobra.Command) {
	cmd.Flags().BoolP("upstream", "u", false, "Select a file for the upstream property")
}

func HandleUpstream(cmd *cobra.Command, vaultDir string) string {
	upstreamFlag, _ := cmd.Flags().GetBool("upstream")

	if upstreamFlag {
		finder := fzf.NewFuzzyFinder(vaultDir, "Select upstream file.")
		upstreamFile, err := finder.Run(false)

		if err != nil {
			fmt.Printf("error selecting upstream file: %s", err)
			os.Exit(1)
		}

		// Extract just the base file name
		baseFileName := filepath.Base(upstreamFile)
		trimmedName := strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName))
		return trimmedName
	}

	return ""
}
