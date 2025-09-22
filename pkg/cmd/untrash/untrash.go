package untrash

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	cmdpkg "github.com/Paintersrp/an/pkg/cmd"
)

func NewCmdUntrash(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "untrash [path]",
		Short: "Restore a note from the trash.",
		Long: heredoc.Doc(`
			This command restores a note from the 'trash' subdirectory.
			Provide the path to the note you want to untrash.

			Example:
			  an untrash /path/to/trashed/note
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				_ = cmd.Help()
				return fmt.Errorf("path argument is required")
			}
			path, err := cmdpkg.ResolveVaultPath(cmd, s, args[0])
			if err != nil {
				return err
			}
			return s.Handler.Untrash(path)
		},
	}

	return cmd
}
