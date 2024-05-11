package taskNewEchoFile

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/zet"
	"github.com/Paintersrp/an/pkg/shared/arg"
	"github.com/Paintersrp/an/pkg/shared/flags"
)

func NewCmdNewEchoFile(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task create-echo [tags] --pin",
		Aliases: []string{"ce"},
		Short:   "Create a new task echo file and optionally pin it.",
		Long:    `Create a new task echo file with a unique incrementing title and optionally pin it using the --pin flag.`,
		Example: "task create-echo 'tag1 tag2' --pin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, s)
		},
	}

	flags.AddPin(cmd)

	return cmd
}

// TODO: Add confirmation logic based on mode?
func run(
	cmd *cobra.Command,
	args []string,
	s *state.State,
) error {
	tags := arg.HandleTags(args)
	rootSubdirFlag := viper.GetString("subdir")
	rootVaultDirFlag := viper.GetString("vaultdir")

	// Get the highest existing increment
	highestIncrement := findHighestIncrement(rootVaultDirFlag, rootSubdirFlag)

	// Generate the next title
	nextTitle := fmt.Sprintf("task-echo-%02d", highestIncrement+1)

	note := zet.NewZettelkastenNote(
		rootVaultDirFlag,
		rootSubdirFlag,
		nextTitle,
		tags,
		[]string{},
		"",
	)

	flags.HandlePin(cmd, s.Config, note, "task", nextTitle)

	zet.StaticHandleNoteLaunch(note, s.Templater, "task-echo", "")

	return nil // no errors
}

func findHighestIncrement(vaultDir, molecule string) int {
	// Regular expression to match titles like "task-echo-01.md"
	re := regexp.MustCompile(`^task-echo-(\d{2})\.md$`)

	// Iterate through existing notes and find the highest increment
	highest := 0
	notes, _ := zet.GetNotesInDirectory(vaultDir, molecule)
	for _, note := range notes {
		match := re.FindStringSubmatch(note)
		if len(match) == 2 {
			increment, _ := strconv.Atoi(match[1])
			if increment > highest {
				highest = increment
			}
		}
	}

	return highest
}
