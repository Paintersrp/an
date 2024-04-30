package day

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/Paintersrp/an/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdDay(
	c *config.Config,
	t *templater.Templater,
) *cobra.Command {
	var index int
	cmd := &cobra.Command{
		Use:     "day [tags] --index --links",
		Aliases: []string{"d"},
		Short:   "Create or open a daily note.",
		Long: `
  This command creates or opens a daily note based on the given index.
  The index can be negative for past days, positive for future days, or zero for today.

  Examples:
  an-cli day --index -1  // Opens yesterday's note
  an-cli day --index +1  // Creates or opens tomorrow's note
  an-cli day             // Opens today's note (default index is 0)
  `,
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, tagsErr := utils.ValidateInput(strings.Join(args, " "))

			if tagsErr != nil {
				fmt.Printf("error processing tags argument: %s", tagsErr)
				os.Exit(1)
			}

			linksFlag, err := cmd.Flags().GetString("links")
			if err != nil {
				fmt.Printf("error retrieving links flag: %s\n", err)
				os.Exit(1)
			}

			links, linksErr := utils.ValidateInput(linksFlag)
			if linksErr != nil {
				fmt.Printf("error processing links flag: %s", linksErr)
				os.Exit(1)
			}

			date := time.Now().AddDate(0, 0, index).Format("20060102")
			tmpl := "day" // Default template for daily notes

			vaultDir := viper.GetString("vaultdir")
			note := zet.NewZettelkastenNote(
				vaultDir,
				"atoms", // default to only atoms for day files, can change later if want to
				fmt.Sprintf("day-%s", date),
				tags,
				links,
			)

			exists, _, existsErr := note.FileExists()
			if existsErr != nil {
				return existsErr
			}

			if exists {
				return note.Open()
			}

			_, createErr := note.Create(tmpl, t)
			if createErr != nil {
				return createErr
			}

			return note.Open()
		},
	}

	cmd.Flags().
		StringP(
			"links",
			"l",
			"",
			"Links for the new note, separated by spaces",
		)

	cmd.Flags().
		IntVarP(&index, "index", "i", 0, "Index for the day relative to today. Can be negative for past days or positive for future days.")
	return cmd
}
