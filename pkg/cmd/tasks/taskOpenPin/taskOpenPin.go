package taskOpenPin

import (
	"errors"
	"fmt"
	"os"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/spf13/cobra"
)

func NewCmdTaskOpenPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open-pin",
		Aliases: []string{"op", "open-p", "o-p"},
		Short:   "Open the pinned file",
		Long:    `OpenPin opens the user's currently pinned file for quick access and editing.`,
		Example: `
    # Opens the currently pinned file open-pin
    # Use the alias 'op', 'o-p', 'open-p', or 'open-pin' to open the pinned file
    an tasks open-pin
    an tasks open-p
    an tasks o-p
    an tasks op
    `,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(c)
		},
	}
	return cmd
}

func run(c *config.Config) error {
	if c.PinnedTaskFile == "" {
		return errors.New("no pinned task file found")
	}
	if _, err := os.Stat(c.PinnedTaskFile); os.IsNotExist(err) {
		return fmt.Errorf("the pinned task file '%s' does not exist", c.PinnedTaskFile)
	}
	return zet.OpenFromPath(c.PinnedTaskFile)
}
