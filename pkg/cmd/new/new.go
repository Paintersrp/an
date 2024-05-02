package new

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/arg"
	"github.com/Paintersrp/an/pkg/flags"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
)

func NewCmdNew(
	c *config.Config,
	t *templater.Templater,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [title] [tags] [--template template_name] [--links link1 link2 ...] [--pin] [--upstream]",
		Aliases: []string{"n"},
		Short:   "Create a new zettelkasten note.",
		Long: heredoc.Doc(`
			The 'new' command creates a new atomic zettelkasten note in your note vault directory.
			Provide a required title argument and an optional tags argument to add tags to the newly created note.
			You can also specify a template, add links, pin the note, or set an upstream file using flags.
		`),
		Example: heredoc.Doc(`
			an new cli-notes "cli notetaking zettel" --links 'zettelkasten cli-moc' --upstream
			an n Tasks -t tasks --pin
		`),
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, c, t)
		},
	}

	flags.AddTemplate(cmd)
	flags.AddLinks(cmd)
	flags.AddUpstream(cmd)
	flags.AddPin(cmd)

	return cmd
}

func run(
	cmd *cobra.Command,
	args []string,
	c *config.Config,
	t *templater.Templater,
) error {
	rootSubdir := viper.GetString("subdir")
	rootVaultDir := viper.GetString("vaultdir")

	title, err := arg.HandleTitle(args)
	if err != nil {
		return err
	}

	tags := arg.HandleTags(args)
	links := flags.HandleLinks(cmd)
	tmpl := flags.HandleTemplate(cmd)
	upstream := flags.HandleUpstream(cmd, rootVaultDir)

	note := zet.NewZettelkastenNote(
		rootVaultDir,
		rootSubdir,
		title,
		tags,
		links,
		upstream,
	)

	conflict := note.HandleConflicts()
	if conflict != nil {
		// HandleConflicts prints feedback if an error is encountered
		return fmt.Errorf("%s", conflict)
	}

	flags.HandlePin(cmd, c, note, "text")
	zet.StaticHandleNoteLaunch(note, t, tmpl)

	return nil // no errors
}
