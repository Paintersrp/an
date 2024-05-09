package new

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/atotto/clipboard"
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
		Use:     "new [title] [tags] [content] [--template template_name] [--links link1 link2 ...] [--pin] [--upstream] [--symlink] [--paste]",
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
		Args: cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, c, t)
		},
	}

	flags.AddTemplate(cmd, "zet")
	flags.AddLinks(cmd)
	flags.AddUpstream(cmd)
	flags.AddPin(cmd)

	cmd.Flags().
		Bool("symlink", false, "Automatically add a symlink to the new note in the current working directory.")

	cmd.Flags().
		Bool("paste", false, "Automatically paste clipboard contents as note content in placeholder.")

	return cmd
}

func run(
	cmd *cobra.Command,
	args []string,
	c *config.Config,
	t *templater.Templater,
) error {
	var content string
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
	createSymlink, err := cmd.Flags().GetBool("symlink")
	if err != nil {
		return err
	}

	paste, err := cmd.Flags().GetBool("paste")
	if err != nil {
		return err
	}

	if paste {
		msg, err := clipboard.ReadAll()
		if err == nil && msg != "" {
			content = msg
		}
	} else {
		content = arg.HandleContent(args)
	}

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

	flags.HandlePin(cmd, c, note, "text", title)
	zet.StaticHandleNoteLaunch(note, t, tmpl, content)

	if createSymlink {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Create the symlink in the current working directory
		symlinkPath := filepath.Join(cwd, filepath.Base(note.GetFilepath()))
		if err := os.Symlink(note.GetFilepath(), symlinkPath); err != nil {
			return fmt.Errorf("failed to create symlink: %s", err)
		}
	}

	return nil // no errors
}
