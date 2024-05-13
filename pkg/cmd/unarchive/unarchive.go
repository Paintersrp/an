package unarchive

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/spf13/cobra"
)

// Not very useful on it's own, but quite handy for scripting
func NewCmdUnarchive(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive [path]",
		Short: "Unarchive a note.",
		Long: heredoc.Doc(`
			This command unarchives a note by moving it from the 'archive' subdirectory.
			Provide the path to the archived note you want to unarchive.

			Example:
			  an unarchive /path/to/archived/note
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				fmt.Println(
					"Please provide the path to the archived note you want to unarchive.",
				)
				return nil
			}
			path := args[0]
			return s.Handler.Unarchive(path)
		},
	}

	return cmd
}
