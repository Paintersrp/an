package pinTable

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/tui/pinList"
)

func Command(s *state.State, pinType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "table",
		Aliases: []string{"t"},
		Short:   "",
		Long:    heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			pinList.Run(s, pinType)
			return nil

		},
	}

	return cmd
}
