package journal

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/journal/entry"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/spf13/cobra"
)

func NewCmdJournal(c *config.Config, t *templater.Templater) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "journal",
		Aliases: []string{"j"},
		Short:   "",
		Long:    heredoc.Doc(``),
		// Run find by default if only a query / no subcommand
		// RunE: openFind.NewCmdOpenFind(c).RunE,
	}

	dayCmd := entry.NewCmdEntry(c, t, "day")
	weekCmd := entry.NewCmdEntry(c, t, "week")
	monthCmd := entry.NewCmdEntry(c, t, "month")

	cmd.AddCommand(dayCmd)
	cmd.AddCommand(weekCmd)
	cmd.AddCommand(monthCmd)

	return cmd
}
