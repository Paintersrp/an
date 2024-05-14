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
		Use:     "new [title] [tags] [content] [--template template_name] [--links link1 link2 ...] [--pin] [--upstream] [--symlink] [--reverse-symlink] [--paste]",
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
	cmd.Flags().
		Bool("reverse-symlink", false, "Create the note in the current working directory and add a symlink in the vault.")

	return cmd
}

func run(
	cmd *cobra.Command,
	args []string,
	s *state.State,
) error {
	subDir := viper.GetString("subdir")
	s.Config.HandleSubdir(subDir)

	vaultDir := viper.GetString("vaultdir")

	title, err := arg.HandleTitle(args)
	if err != nil {
		return err
	}

	tags := arg.HandleTags(args)
	links := flags.HandleLinks(cmd)
	tmpl := flags.HandleTemplate(cmd)
	upstream := flags.HandleUpstream(cmd, vaultDir)

	createSymlink, err := cmd.Flags().GetBool("symlink")
	if err != nil {
		return err
	}

	reverseSymlink, err := cmd.Flags().GetBool("reverse-symlink")
	if err != nil {
		return err
	}

	if createSymlink && reverseSymlink {
		return fmt.Errorf(
			"cannot use both --symlink and --reverse-symlink flags simultaneously",
		)
	}

	paste, err := flags.HandlePaste(cmd)
	if err != nil {
		return err
	}

	var content string
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

	var note *zet.ZettelkastenNote

	if createSymlink {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		note = zet.NewZettelkastenNote(vaultDir, subDir, title, tags, links, upstream)

		symlinkPath := filepath.Join(cwd, filepath.Base(note.GetFilepath()))
		if err := os.Symlink(note.GetFilepath(), symlinkPath); err != nil {
			return fmt.Errorf("failed to create symlink: %s", err)
		}

	} else {
		if reverseSymlink {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			note = zet.NewZettelkastenNote(cwd, "", title, tags, links, upstream)

			fmt.Printf("Note path: %s", note.GetFilepath())

			reverseSymlinkPath := filepath.Join(vaultDir, subDir, filepath.Base(note.GetFilepath()))
			if err := os.Symlink(note.GetFilepath(), reverseSymlinkPath); err != nil {
				fmt.Println(err)
				os.Exit(1)
				return fmt.Errorf("failed to create symlink: %s", err)
			}

			fmt.Printf("Reverse Path: %s", reverseSymlinkPath)
		} else {
			note = zet.NewZettelkastenNote(vaultDir, subDir, title, tags, links, upstream)
		}
	}

	conflict := note.HandleConflicts()
	if conflict != nil {
		return fmt.Errorf("%s", conflict)
	}

	flags.HandlePin(cmd, s.Config, note, "text", title)
	zet.StaticHandleNoteLaunch(note, s.Templater, tmpl, content)

	if createSymlink {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		symlinkPath := filepath.Join(cwd, filepath.Base(note.GetFilepath()))
		if err := os.Symlink(note.GetFilepath(), symlinkPath); err != nil {
			return fmt.Errorf("failed to create symlink: %s", err)
		}

	}
	return nil
}
