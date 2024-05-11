package new

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/zet"
	"github.com/Paintersrp/an/pkg/shared/arg"
	"github.com/Paintersrp/an/pkg/shared/flags"
)

func NewCmdNew(s *state.State) *cobra.Command {
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
			return run(cmd, args, s)
		},
	}

	flags.AddTemplate(cmd, "zet")
	flags.AddLinks(cmd)
	flags.AddUpstream(cmd)
	flags.AddPin(cmd)
	flags.AddPaste(cmd)

	cmd.Flags().
		Bool("symlink", false, "Automatically add a symlink to the new note in the current working directory.")

	return cmd
}

func run(
	cmd *cobra.Command,
	args []string,
	s *state.State,
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

	paste, err := flags.HandlePaste(cmd)
	if err != nil {
		return err
	}

	if paste {
		msg, err := clipboard.ReadAll()
		if err == nil && msg != "" {
			content = msg
		}
	} else {
		if len(args) < 1 {
			content = arg.HandleContent(args)
		}
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

	flags.HandlePin(cmd, s.Config, note, "text", title)
	zet.StaticHandleNoteLaunch(note, s.Templater, tmpl, content)

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

	return nil
}
