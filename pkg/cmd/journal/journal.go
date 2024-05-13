package journal

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/journal/entry"
)

func NewCmdJournal(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "journal",
		Aliases: []string{"j"},
		Short:   " The journal command creates or opens a note based on the given index. You can specify whether it’s for a day, week, month, or year. Additionally, you can add links to your note using the --links flag.",
		Long: heredoc.Doc(`
This command creates or opens a note based on the given index. The index can be negative for past notes (e.g., days, weeks) or positive for future notes. A zero index corresponds to today. You can also add links to your note using the --links flag.

Examples:
  an j day --index -1  // Opens the previous day's note
  an j week --index +1  // Creates or opens the next week's note
  an j month             // Opens the current month's note (default index is 0)
  an j year              // Opens the current year's note with links
      `),
	}

	dayCmd := entry.NewCmdEntry(s, "day")
	weekCmd := entry.NewCmdEntry(s, "week")
	monthCmd := entry.NewCmdEntry(s, "month")
	yearCmd := entry.NewCmdEntry(s, "year")

	cmd.AddCommand(dayCmd)
	cmd.AddCommand(weekCmd)
	cmd.AddCommand(monthCmd)
	cmd.AddCommand(yearCmd)

	return cmd
}
