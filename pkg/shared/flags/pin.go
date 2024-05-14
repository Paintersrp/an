package flags

import (
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/note"
)

func AddPin(cmd *cobra.Command) {
	cmd.Flags().BoolP("pin", "p", false, "Pin the newly created file")
}

func HandlePin(
	cmd *cobra.Command,
	c *config.Config,
	n *note.ZettelkastenNote,
	pinType string,
	name string,
) {
	pinFlag, _ := cmd.Flags().GetBool("pin")

	if pinFlag {
		c.ChangePin(n.GetFilepath(), pinType, name)
	}
}
