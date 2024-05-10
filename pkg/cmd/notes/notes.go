package notes

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	v "github.com/Paintersrp/an/internal/views"
	"github.com/Paintersrp/an/tui/notes"
)

// TODO: Could also allow shorthands for view flag with parsing

func NewCmdNotes(s *state.State) *cobra.Command {
	var viewFlag string

	cmd := &cobra.Command{
		Use:     "notes",
		Aliases: []string{"n"},
		Short:   "",
		Long:    heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(s, s.Views, viewFlag)
		},
	}

	cmd.Flags().StringVarP(&viewFlag, "view", "v", "default", "Select initial view")
	return cmd
}

func run(
	s *state.State,
	views map[string]v.View,
	viewFlag string,
) error {
	// Pass modeConfig to your list model or wherever it's needed
	if err := notes.Run(s, views, viewFlag); err != nil {
		return err
	}
	return nil
}
