package new

import (
	"fmt"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/Paintersrp/an/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdNew(
	c *config.Config,
	t *templater.Templater,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [title] [tags]",
		Aliases: []string{"n"},
		Short:   "Create a new zettelkasten note.",
		Long: `
  This command creates a new atomic kettelkasten note into your note vault directory.
  It takes a required title argument and an optional tags argument to quickly add tags to the newly made note.

              [title]  [tags]
  an new robotics-class "robotics science class study-notes"
  `,
		Example: "atomic new cli-notes 'cli go zettelkasten notetaking learn'",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf(
					"error: No title given. Try again with 'an new [title]'",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, c, t)
		},
	}

	cmd.Flags().
		StringP(
			"template",
			"t",
			"zet",
			"Specify the template to use (default is 'zet'). Available templates: daily, roadmap, zet",
		)
	viper.BindPFlag(
		"template",
		cmd.Flags().Lookup("template"),
	)

	cmd.Flags().
		StringP(
			"links",
			"l",
			"",
			"Links for the new note, separated by spaces",
		)
	viper.BindPFlag("links", cmd.Flags().Lookup("links"))

	// Add the --pin flag
	cmd.Flags().BoolP("pin", "p", false, "Pin the newly created file")

	return cmd
}

func run(
	cmd *cobra.Command,
	args []string,
	c *config.Config,
	t *templater.Templater,
) error {
	title := args[0]
	tags := handleTags(args)
	links := handleLinks(cmd)
	tmpl := handleTemplate(cmd)
	rootMoleculeFlag := viper.GetString("molecule")
	rootVaultDirFlag := viper.GetString("vaultdir")

	note := zet.NewZettelkastenNote(
		rootVaultDirFlag,
		rootMoleculeFlag,
		title,
		tags,
		links,
	)

	handleConflicts(note)

	// Check if the --pin flag is set
	pinFlag, _ := cmd.Flags().GetBool("pin")
	if pinFlag {
		// Update the config's pinned file to the new file path
		c.ChangePin(note.GetFilepath(), "text")
	}

	zet.StaticHandleNoteLaunch(note, t, tmpl)

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

func handleLinks(cmd *cobra.Command) []string {
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

	return links
}

func handleTemplate(cmd *cobra.Command) string {
	tmpl, err := cmd.Flags().GetString("template")
	if err != nil {
		fmt.Printf("error retrieving template flag: %s\n", err)
		os.Exit(1)
	}

	if _, ok := templater.AvailableTemplates[tmpl]; !ok {
		// Create a slice to hold the keys from AvailableTemplates.
		var templateNames []string
		for name := range templater.AvailableTemplates {
			templateNames = append(templateNames, name)
		}

		// Join the template names into a single string separated by commas.
		availableTemplateNames := strings.Join(templateNames, ", ")

		fmt.Printf(
			"invalid template specified. Available templates are: %s",
			availableTemplateNames,
		)
		os.Exit(1)
	}

	return tmpl
}

func handleConflicts(note *zet.ZettelkastenNote) {
	exists, _, existsErr := note.FileExists()
	if existsErr != nil {
		fmt.Printf("error processing note file: %s", existsErr)
		os.Exit(1)
	}

	if exists {
		fmt.Println("error: Note with given title already exists in the vault directory.")
		fmt.Println(
			"hint: Try again with a new title, or run again with either an overwrite (-o) flag or an increment (-i) flag",
		)
		os.Exit(1)
	}
}
