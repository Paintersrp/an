package openPin

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/spf13/cobra"
)

func NewCmdOpenPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open-pin",
		Aliases: []string{"op"},
		Short:   "",
		Long:    ``,
		Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			zet.OpenFromPath(c.PinnedFile)
			return nil
		},
	}
	return cmd
}
