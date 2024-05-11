package symlink

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/fzf"
	"github.com/Paintersrp/an/internal/state"
)

func NewCmdSymlink(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "symlink [query]",
		Aliases: []string{"sl", "sym"},
		Short:   "Symlink a zettelkasten note to the current working directory.",
		Long: heredoc.Doc(`
			This command symlinks a zettelkasten note to the current working directory.
			It takes one optional argument for a note title, the note to be symlinked.
			If no title is given, the vault directory files will be displayed
			with a fuzzy finder and file preview for selection.

			Examples:
			  an symlink cli-notes  // Fuzzyfind with query and symlink
			  an sym linux          // Fuzzyfind with query and symlink
			  an sl                 // Fuzzyfind no query and symlink
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return symlink(args, s)
		},
	}

	return cmd
}

func symlink(args []string, s *state.State) error {
	finder := fzf.NewFuzzyFinder(s.Vault, "Select file to symlink.")

	var (
		selectedFile string
		selectError  error
	)
	if len(args) == 0 {
		selectedFile, selectError = finder.Run(false)
	} else {
		selectedFile, selectError = finder.RunWithQuery(args[0], false)
	}

	if selectError != nil {
		return fmt.Errorf("file selection error: %s", selectError)
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create the symlink in the current working directory
	symlinkPath := filepath.Join(cwd, filepath.Base(selectedFile))
	return os.Symlink(selectedFile, symlinkPath)
}
