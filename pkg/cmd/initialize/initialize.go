package initialize

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/tui/initialize"
)

func NewCmdInit(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "initialize",
		Aliases: []string{"i", "init"},
		Short:   "initialize atomic-notes-cli",
		Long: heredoc.Doc(`
			Launch your atomic-notes-cli journey with this initialization command.
			It will guide you through a series of prompts to tailor the CLI to your preferences,
			ensuring a personalized and optimized experience. From setting up your default editor
			to configuring file paths, this interactive setup prepares your environment for seamless note-taking.
		`),
		Example: heredoc.Doc(`
			an initialize
			an init
			an i
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initialize.Run(s); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
