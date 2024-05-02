package flags

import (
	"fmt"
	"os"
	"strings"

	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AddTemplate(cmd *cobra.Command) {
	cmd.Flags().
		StringP(
			"template",
			"t",
			"zet",
			"Specify the template to use (default is 'zet'). Available templates: daily, roadmap, zet",
		)
	viper.BindPFlag(
		"template",
		cmd.Flags().Lookup("template"),
	)
}

func HandleTemplate(cmd *cobra.Command) string {
	tmpl, err := cmd.Flags().GetString("template")
	if err != nil {
		fmt.Printf("error retrieving template flag: %s\n", err)
		os.Exit(1)
	}

	if _, ok := templater.AvailableTemplates[tmpl]; !ok {
		// Create a slice to hold the keys from AvailableTemplates.
		var templateNames []string
		for name := range templater.AvailableTemplates {
			templateNames = append(templateNames, name)
		}

		// Join the template names into a single string separated by commas.
		availableTemplateNames := strings.Join(templateNames, ", ")

		fmt.Printf(
			"invalid template specified. Available templates are: %s",
			availableTemplateNames,
		)
		os.Exit(1)
	}

	return tmpl
}
