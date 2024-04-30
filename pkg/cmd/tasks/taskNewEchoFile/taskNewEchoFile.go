package taskNewEchoFile

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/Paintersrp/an/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdNewEchoFile(
	c *config.Config,
	t *templater.Templater,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task create-echo [tags] --pin",
		Aliases: []string{"ce"},
		Short:   "Create a new task echo file and optionally pin it.",
		Long:    `Create a new task echo file with a unique incrementing title and optionally pin it using the --pin flag.`,
		Example: "task create-echo 'tag1 tag2' --pin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, t, c)
		},
	}
	// Add the --pin flag
	cmd.Flags().BoolP("pin", "p", false, "Pin the newly created task echo file")

	return cmd
}

// TODO: Add confirmation logic based on mode?
func run(
	cmd *cobra.Command,
	args []string,
	t *templater.Templater,
	c *config.Config,
) error {
	tags := handleTags(args)
	rootMoleculeFlag := viper.GetString("molecule")
	rootVaultDirFlag := viper.GetString("vaultdir")

	// Get the highest existing increment
	highestIncrement := findHighestIncrement(rootVaultDirFlag, rootMoleculeFlag)

	// Generate the next title
	nextTitle := fmt.Sprintf("task-echo-%02d", highestIncrement+1)

	note := zet.NewZettelkastenNote(
		rootVaultDirFlag,
		rootMoleculeFlag,
		nextTitle,
		tags,
		[]string{},
	)

	// Check if the --pin flag is set
	pinFlag, _ := cmd.Flags().GetBool("pin")
	if pinFlag {
		// Update the config's pinned task file to the new file path
		c.ChangePin(note.GetFilepath(), "task")
	}

	zet.StaticHandleNoteLaunch(note, t, "task-echo")

	return nil // no errors
}

func handleTags(args []string) []string {
	var (
		tags    []string
		tagsErr error
	)

	if len(args) > 1 {
		tags, tagsErr = utils.ValidateInput(args[1])

		if tagsErr != nil {
			fmt.Printf("error processing tags argument: %s", tagsErr)
			os.Exit(1)
		}

	}

	return tags
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
