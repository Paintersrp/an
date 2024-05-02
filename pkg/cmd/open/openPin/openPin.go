package openPin

import (
	"errors"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
)

func NewCmdOpenPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pin",
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
