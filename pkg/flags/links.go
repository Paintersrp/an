package flags

import (
	"fmt"
	"os"

	"github.com/Paintersrp/an/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AddLinks(cmd *cobra.Command) {
	cmd.Flags().
		StringP(
			"links",
			"l",
			"",
			"Links for the new note, separated by spaces",
		)
	viper.BindPFlag("links", cmd.Flags().Lookup("links"))
}

func HandleLinks(cmd *cobra.Command) []string {
	linksFlag, err := cmd.Flags().GetString("links")
	if err != nil {
		fmt.Printf("error retrieving links flag: %s\n", err)
		os.Exit(1)
	}

	links, err := utils.ValidateInput(linksFlag)
	if err != nil {
		fmt.Printf("error processing links flag: %s", err)
		os.Exit(1)
	}

	return links
}
