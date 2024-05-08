package flags

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/spf13/cobra"
)

func AddPin(cmd *cobra.Command) {
	cmd.Flags().BoolP("pin", "p", false, "Pin the newly created file")
}

func HandlePin(
	cmd *cobra.Command,
	c *config.Config,
	note *zet.ZettelkastenNote,
	pinType string,
	name string,
) {
	pinFlag, _ := cmd.Flags().GetBool("pin")

	if pinFlag {
		c.ChangePin(note.GetFilepath(), pinType, name)
	}
}
