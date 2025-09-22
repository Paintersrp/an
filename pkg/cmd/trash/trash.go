package trash

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	cmdpkg "github.com/Paintersrp/an/pkg/cmd"
	"github.com/spf13/cobra"
)

func NewCmdTrash(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trash [path]",
		Short: "Move a note to the trash.",
		Long: heredoc.Doc(`
			This command moves a note to the 'trash' subdirectory.
			Provide the path to the note you want to move to the trash.

			Example:
			  an trash /path/to/note
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
			return s.Handler.Trash(path)
		},
	}

	return cmd
}
