package echo

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdEcho(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "echo [message]",
		Short: "Append a message to the pinned file.",
		Long: heredoc.Doc(`
			The echo command appends a message to the pinned file.
			If no file is pinned, it returns an error.

			Examples:
			  an echo "This is a message."
			  an echo "Add this to the pinned file."
		`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, c)
		},
	}
	return cmd
}

func run(args []string, c *config.Config) error {
	message := strings.Join(args, " ")

	if c.PinnedFile == "" {
		return errors.New(
			"no file pinned. Use the pin command to pin a file first",
		)
	}
	// Append the message to the pinned file
	file, err := os.OpenFile(
		c.PinnedFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(message + "\n"); err != nil {
		return err
	}
	fmt.Println("Message appended to the pinned file.")
	return nil
}
