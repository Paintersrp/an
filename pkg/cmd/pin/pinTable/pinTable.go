package pinTable

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/tui/pinList"
)

func Command(c *config.Config, pinType string) *cobra.Command {

	cmd := &cobra.Command{
		Use:     "table",
		Aliases: []string{"t"},
		Short:   "",
		Long:    heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			pinList.Run(c, pinType)
			return nil

		},
	}

	return cmd
}
