package notes

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/tui/notes"
	v "github.com/Paintersrp/an/internal/views"
)

// TODO: Could also allow shorthands for view flag with parsing

func NewCmdNotes(s *state.State) *cobra.Command {
	var viewFlag string
	cmd := &cobra.Command{
		Use:     "notes",
		Aliases: []string{"n"},
		Short:   "Manage and interact with your notes vault",
		Long: heredoc.Doc(`
            The 'notes' command provides a Terminal User Interface (TUI) for managing
            and interacting with your notes vault. This powerful interface allows you
            to perform various operations on your notes, including:

            - Archiving and unarchiving notes
            - Trashing and untrashing notes
            - Deleting notes
            - Opening notes in your configured editor
            - Editing note metadata and contents
            - Fuzzy finding notes by title or content
            - Viewing and managing orphan notes
            - Viewing and managing unfulfilled notes
            - And more!

            Use the '--view' flag to select the initial view you want to start with.
            The available views are:

            - 1. 'default': The default view showing all your notes.
            - 2. 'archive': A view displaying archived notes.
            - 3. 'orphan': A view for finding and managing orphan (notes with no links to other notes).
            - 4. 'unfulfilled': A view focused on notes marked as unfulfilled.
            - 5. 'trash': A view showing trashed notes.

            Navigate through the TUI using your keyboard, and follow the on-screen
            instructions to perform various actions on your notes.
        `),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(s, s.Views, viewFlag)
		},
	}

	cmd.Flags().StringVarP(&viewFlag, "view", "v", "default", "Select initial view")
	return cmd
}

func run(s *state.State, views map[string]v.View, viewFlag string) error {
	// Pass modeConfig to your list model or wherever it's needed
	if err := notes.Run(s, views, viewFlag); err != nil {
		return err
	}
	return nil
}
