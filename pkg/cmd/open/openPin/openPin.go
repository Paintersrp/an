package openPin

import (
	"errors"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/zet"
	"github.com/Paintersrp/an/pkg/shared/flags"
)

func NewCmdOpenPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pin -n {pin-name}",
		Aliases: []string{"p"},
		Short:   "Open the pinned file",
		Long: heredoc.Doc(`
			OpenPin provides quick access to your most important file. With this command,
			you can immediately open the file you've pinned, ensuring that your key notes
			and information are always at your fingertips. Customize your experience with
			various flags to streamline your workflow.
		`),
		Example: heredoc.Doc(`
      an open pin
      an o p
    `),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, c)
		},
	}
	flags.AddName(cmd, "Named pin to target")
	return cmd
}

func run(cmd *cobra.Command, c *config.Config) error {
	name, err := flags.HandleName(cmd)
	if err != nil {
		return err
	}

	var targetPin string
	if name != "" {
		if c.NamedPins[name] == "" {
			return fmt.Errorf("no pinned file found")
		}
		targetPin = c.NamedPins[name]
	} else {
		if c.PinnedFile == "" {
			return errors.New("no pinned file found")
		}
		targetPin = c.PinnedFile
	}

	if _, err := os.Stat(targetPin); os.IsNotExist(err) {
		return fmt.Errorf("the pinned file '%s' does not exist", targetPin)
	}
	return zet.OpenFromPath(targetPin, false)

}
