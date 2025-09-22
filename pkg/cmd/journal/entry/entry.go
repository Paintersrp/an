package entry

import (
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/pkg/shared/flags"
	"github.com/Paintersrp/an/utils"
)

var readClipboard = clipboard.ReadAll

// TODO: adding links/tags/content after note already exists?

func NewCmdEntry(
	s *state.State,
	templateType string, // Accepts "day", "week", "month", or "year"
) *cobra.Command {
	var index int

	cmd := &cobra.Command{
		Use: fmt.Sprintf(
			"%s [tags] [content] [--index index] [--links link1 link2 ...] [--paste]",
			templateType,
		),
		Aliases: []string{strings.ToLower(templateType[:1])},
		Short:   fmt.Sprintf("Create or open a %s note.", templateType),
		Long: heredoc.Doc(fmt.Sprintf(
			`
			This command creates or opens a %s note based on the given index.
			The index can be negative for past %ss, positive for future %ss, or zero for today.
			You can also add links to your %s note using the --links flag.

			Examples:
			  an j %s --index -1  // Opens previous %s
			  an j %s --index +1  // Creates or opens the next %s
			  an j %s             // Opens current's %s (default index is 0)
			  an j %s             // Opens current's %s with links
		`,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
			templateType,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, s.Templater, index, templateType)
		},
	}

	flags.AddLinks(cmd)
	flags.AddPaste(cmd)
	cmd.Flags().
		IntVarP(&index, "index", "i", 0, fmt.Sprintf("Index for the %s relative to today. Can be negative for past %ss or positive for future %ss.", templateType, templateType, templateType))

	return cmd
}

func run(
	cmd *cobra.Command,
	args []string,
	t *templater.Templater,
	index int,
	templateType string,
) error {
	tagInput := ""
	if len(args) > 0 {
		tagInput = args[0]
	}

	tags, err := utils.ValidateInput(tagInput)
	if err != nil {
		fmt.Printf("error processing tags argument: %s", err)
		os.Exit(1)
		return err
	}

	content := ""
	paste, err := flags.HandlePaste(cmd)
	if err != nil {
		return err
	}

	if paste {
		msg, err := readClipboard()
		if err == nil && msg != "" {
			content = msg
		}
	} else {
		if len(args) >= 2 {
			content = args[1]
		}
	}

	links := flags.HandleLinks(cmd)
	date := utils.GenerateDate(index, templateType)
	vaultDir := viper.GetString("vaultdir")

	n := note.NewZettelkastenNote(
		vaultDir,
		"atoms",
		fmt.Sprintf("%s-%s", templateType, date),
		tags,
		links,
		"",
	)

	exists, _, err := n.FileExists()
	if err != nil {
		return err
	}

	if exists {
		return n.Open()
	}

	_, createErr := n.Create(templateType, t, content)
	if createErr != nil {
		return createErr
	}

	return n.Open()
}
