package taskOpenPin

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/zet"
	"github.com/Paintersrp/an/pkg/shared/flags"
)

func NewCmdTaskOpenPin(s *state.State) *cobra.Command {
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
			return run(cmd, s)
		},
	}

	flags.AddName(cmd, "Name for new saved pin")
	return cmd
}

func run(cmd *cobra.Command, s *state.State) error {
	name, err := flags.HandleName(cmd)
	if err != nil {
		return err
	}

	var targetPin string
	if name != "" {
		if s.Config.NamedTaskPins[name] == "" {
			return fmt.Errorf("no pinned task file found")
		}
		targetPin = s.Config.NamedTaskPins[name]
	} else {
		if s.Config.PinnedTaskFile == "" {
			return errors.New("no pinned task file found")
		}
		targetPin = s.Config.PinnedTaskFile
	}

	if _, err := os.Stat(targetPin); os.IsNotExist(err) {
		return fmt.Errorf("the pinned task file '%s' does not exist", targetPin)
	}
	return zet.OpenFromPath(targetPin, false)
}
