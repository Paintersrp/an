package pinOpen

import (
	"errors"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/zet"
	"github.com/Paintersrp/an/pkg/shared/flags"
)

func NewCmdPinOpen(s *state.State, pinType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open -n {pin-name}",
		Aliases: []string{"o"},
		Short:   "Open the pinned file",
		Long: heredoc.Doc(`
			Pin Open provides quick access to your most important file. With this command,
			you can immediately open the file you've pinned, ensuring that your key notes
			and information are always at your fingertips. Customize your experience with
			various flags to streamline your workflow.
		`),
		Example: heredoc.Doc(`
      an pin open
      an p o
    `),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, s, pinType)
		},
	}
	flags.AddName(cmd, "Named pin to target")
	return cmd
}

func run(cmd *cobra.Command, s *state.State, pinType string) error {
	name, err := flags.HandleName(cmd)
	if err != nil {
		return err
	}

	var targetPin string
	if name != "" {
		switch pinType {
		case "text":
			if s.Config.NamedPins[name] == "" {
				return fmt.Errorf("no pinned file found")
			}
			targetPin = s.Config.NamedPins[name]
		case "task":
			if s.Config.NamedTaskPins[name] == "" {
				return fmt.Errorf("no pinned file found")
			}
			targetPin = s.Config.NamedTaskPins[name]
		}
	} else {
		switch pinType {
		case "text":
			if s.Config.PinnedFile == "" {
				return errors.New("no pinned file found")
			}
			targetPin = s.Config.PinnedFile

		case "task":
			if s.Config.PinnedTaskFile == "" {
				return errors.New("no pinned file found")
			}
			targetPin = s.Config.PinnedTaskFile
		}
	}

	if _, err := os.Stat(targetPin); os.IsNotExist(err) {
		return fmt.Errorf("the pinned file '%s' does not exist", targetPin)
	}
	return zet.OpenFromPath(targetPin, false)

}
