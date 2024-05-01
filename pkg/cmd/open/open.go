package open

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/fzf"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdOpen(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open [query]",
		Aliases: []string{"o"},
		Short:   "Open a zettelkasten note.",
		Long: `This command opens a zettelkasten note with nvim, ready for editing.
  It takes one optional argument for a note title, the note to be opened.
  If no title is given, the vault directory files will be displayed
  with a fuzzy finder and file preview for selection, which will pipe into the configured editor to open.`,
		Example: "atomic open cli-notes or atomic open",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			vaultDir := viper.GetString("vaultDir")
			vaultFlag, _ := cmd.Flags().GetBool("vault")

			if vaultFlag {
				zet.OpenFromPath(vaultDir)
			} else {
				finder := fzf.NewFuzzyFinder(vaultDir, "Select file to open.")

				if len(args) == 0 {
					finder.Run(true)
				} else {
					finder.RunWithQuery(args[0], true)
				}
			}
		},
	}

	cmd.Flags().BoolP("vault", "v", false, "Open the vault directory directly")
	return cmd
}
