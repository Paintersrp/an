package tags

import (
	"fmt"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/parser"
	"github.com/spf13/cobra"
)

func NewCmdTags(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "Display a table of tags",
		Long: `The tags command parses tags from notes and displays them in a sortable and navigable table,
allowing for quick access and organization of notes by their associated tags.`,
		Example: `
    # Display a table of tags
    an-cli tags
    `,
		Run: func(cmd *cobra.Command, args []string) {
			p := parser.NewParser(c.VaultDir)

			if err := p.Walk(); err != nil {
				fmt.Println("Error:", err)
				return
			}

			p.ShowTagTable()
		},
	}

	return cmd
}
