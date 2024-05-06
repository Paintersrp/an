package notes

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/tui/notes"
	"github.com/spf13/cobra"
)

var modeFlag string

func NewCmdNotes(c *config.Config) *cobra.Command {
	modes := notes.GenerateModes(c.VaultDir)

	cmd := &cobra.Command{
		Use:     "notes",
		Aliases: []string{"n"},
		Short:   "",
		Long:    heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(c, modes)
		},
	}

	cmd.Flags().StringVarP(&modeFlag, "mode", "m", "default", "Select mode of operation")
	return cmd
}

func run(c *config.Config, modes map[string]notes.ModeConfig) error {
	// Pass modeConfig to your list model or wherever it's needed
	if err := notes.Run(c, modes, modeFlag); err != nil {
		return err
	}
	return nil
}
