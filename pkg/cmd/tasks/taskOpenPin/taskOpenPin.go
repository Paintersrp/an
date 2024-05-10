package taskOpenPin

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/fs/zet"
	"github.com/Paintersrp/an/internal/config"
)

func NewCmdTaskOpenPin(c *config.Config) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "open-pin -n {pin-name}",
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
			return run(c, name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Save as a named pin")
	return cmd
}

func run(c *config.Config, name string) error {
	var targetPin string
	if name != "" {
		if c.NamedTaskPins[name] == "" {
			return fmt.Errorf("no pinned task file found")
		}
		targetPin = c.NamedTaskPins[name]
	} else {
		if c.PinnedTaskFile == "" {
			return errors.New("no pinned task file found")
		}
		targetPin = c.PinnedTaskFile
	}

	if _, err := os.Stat(targetPin); os.IsNotExist(err) {
		return fmt.Errorf("the pinned task file '%s' does not exist", targetPin)
	}
	return zet.OpenFromPath(targetPin)
}
