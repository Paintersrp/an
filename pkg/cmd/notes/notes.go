package notes

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/fs/templater"
	"github.com/Paintersrp/an/internal/config"
	v "github.com/Paintersrp/an/internal/views"
	"github.com/Paintersrp/an/tui/notes"
)

// TODO: Could also allow shorthands for view flag with parsing

func NewCmdNotes(c *config.Config, t *templater.Templater) *cobra.Command {
	var viewFlag string
	views := v.GenerateViews(c.VaultDir)

	cmd := &cobra.Command{
		Use:     "notes",
		Aliases: []string{"n"},
		Short:   "",
		Long:    heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(c, t, views, viewFlag)
		},
	}

	cmd.Flags().StringVarP(&viewFlag, "view", "v", "default", "Select initial view")
	return cmd
}

func run(
	c *config.Config,
	t *templater.Templater,
	views map[string]v.View,
	viewFlag string,
) error {
	// Pass modeConfig to your list model or wherever it's needed
	if err := notes.Run(c, t, views, viewFlag); err != nil {
		return err
	}
	return nil
}
