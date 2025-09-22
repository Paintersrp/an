package unarchive

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	cmdpkg "github.com/Paintersrp/an/pkg/cmd"
	"github.com/spf13/cobra"
)

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
				_ = cmd.Help()
				return fmt.Errorf("path argument is required")
			}
			path, err := cmdpkg.ResolveVaultPath(cmd, s, args[0])
			if err != nil {
				return err
			}
			return s.Handler.Unarchive(path)
		},
	}

	return cmd
}
