package tags

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/parser"
	"github.com/Paintersrp/an/internal/state"
)

func NewCmdTags(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "Display a table of tags",
		Long: `The tags command parses tags from notes and displays them in a sortable and navigable table,
allowing for quick access and organization of notes by their associated tags.`,
		Example: `
    # Display a table of tags
    an-cli tags
    `,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(s)
		},
	}

	return cmd
}

func run(s *state.State) error {
	p := parser.NewParser(s.Vault)

	if err := p.Walk(); err != nil {
		fmt.Println("Error:", err)
		return err
	}

	p.ShowTagTable()
	return nil
}
