package echo

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdEcho(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "echo [message]",
		Short: "Append a message to the pinned file.",
		Long: `The echo command appends a message to the pinned file.
If no file is pinned, it returns an error.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Message validation
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
		},
	}
	return cmd
}
