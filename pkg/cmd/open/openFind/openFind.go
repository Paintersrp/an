package openFind

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/fzf"
)

func NewCmdOpenFind(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "find [query]",
		Aliases: []string{"f"},
		Short:   "Open a zettelkasten note with fuzzyfinding.",
		Long: heredoc.Doc(`
			This command opens a zettelkasten note with nvim, ready for editing.
			It takes one optional argument for a note title, the note to be opened.
			If no title is given, the vault directory files will be displayed
			with a fuzzy finder and file preview for selection, which will pipe into the configured editor to open.

			Examples:
			  an open find cli-notes  // Fuzzyfind with query
        an o f linux            // Fuzzyfind with query
        an o f                  // Fuzzyfind no query
		`),
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args)
		},
	}

	return cmd
}

func run(args []string) error {
	vaultDir := viper.GetString("vaultDir")
	finder := fzf.NewFuzzyFinder(vaultDir, "Select file to open.")

	if len(args) == 0 {
		finder.Run(true)
	} else {
		finder.RunWithQuery(args[0], true)
	}

	return nil
}
