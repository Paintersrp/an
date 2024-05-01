package openPin

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
)

func NewCmdOpenPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open-pin",
		Aliases: []string{"op, open-p, o-p"},
		Short:   "Open the pinned file",
		Long:    `OpenPin opens the user's currently pinned file for quick access and editing.`,
		Example: `
    # Opens the currently pinned file open-pin
    # Use the alias 'op', 'o-p', 'open-p', or 'open-pin' to open the pinned file
    an open-pin
    an open-p
    an o-p
    an op
    `,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(c)
		},
	}
	return cmd
}

func run(c *config.Config) error {
	if c.PinnedFile == "" {
		return errors.New("no pinned file found")
	}
	if _, err := os.Stat(c.PinnedFile); os.IsNotExist(err) {
		return fmt.Errorf("the pinned file '%s' does not exist", c.PinnedFile)
	}
	return zet.OpenFromPath(c.PinnedFile)
}
