package archive

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/spf13/cobra"
)

func NewCmdArchive(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive [path]",
		Short: "Archive a note.",
		Long: heredoc.Doc(`
			This command archives a note by moving it to the 'archive' subdirectory.
			Provide the path to the note you want to archive.

			Example:
			  an archive /path/to/note
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				fmt.Println("Please provide the path to the note you want to archive.")
				return nil
			}
			path := args[0]
			return s.Handler.Archive(path)
		},
	}

	return cmd
}
