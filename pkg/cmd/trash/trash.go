package trash

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/utils"
)

// Not very useful on it's own, but quite handy for scripting
func NewCmdTrash(
	c *config.Config,
) *cobra.Command {
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
				fmt.Println(
					"Please provide the path to the note you want to move to the trash.",
				)
				return nil
			}
			path := args[0]
			return utils.Trash(path, c)
		},
	}

	return cmd
}
