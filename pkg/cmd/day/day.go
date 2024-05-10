package day

import (
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/fs/templater"
	"github.com/Paintersrp/an/fs/zet"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/flags"
	"github.com/Paintersrp/an/utils"
)

func NewCmdDay(
	c *config.Config,
	t *templater.Templater,
) *cobra.Command {
	var index int
	cmd := &cobra.Command{
		Use:     "day [tags] [--index index] [--links link1 link2 ...]",
		Aliases: []string{"d"},
		Short:   "Create or open a daily note.",
		Long: heredoc.Doc(`
			This command creates or opens a daily note based on the given index.
			The index can be negative for past days, positive for future days, or zero for today.
			You can also add links to your daily note using the --links flag.

			Examples:
			  an day --index -1  // Opens yesterday's note
			  an day --index +1  // Creates or opens tomorrow's note
			  an day             // Opens today's note (default index is 0)
			  an day --links 'Vacation' // Opens today's note with links
		`), RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, t, index)
		},
	}

	flags.AddLinks(cmd)
	cmd.Flags().
		IntVarP(&index, "index", "i", 0, "Index for the day relative to today. Can be negative for past days or positive for future days.")
	return cmd
}

func run(cmd *cobra.Command, args []string, t *templater.Templater, index int) error {
	tags, err := utils.ValidateInput(strings.Join(args, " "))

	if err != nil {
		fmt.Printf("error processing tags argument: %s", err)
		os.Exit(1)
		return err
	}

	links := flags.HandleLinks(cmd)
	date := utils.GenerateDate(index, "day")
	vaultDir := viper.GetString("vaultdir")

	note := zet.NewZettelkastenNote(
		vaultDir,
		"atoms",
		fmt.Sprintf("day-%s", date),
		tags,
		links,
		"",
	)

	exists, _, err := note.FileExists()
	if err != nil {
		return err
	}

	if exists {
		return note.Open()
	}

	// TODO: Content instead of "" ?
	_, createErr := note.Create("day", t, "")
	if createErr != nil {
		return createErr
	}

	return note.Open()
}
