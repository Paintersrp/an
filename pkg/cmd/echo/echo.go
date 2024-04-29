/*
Copyright © 2024 Ryan Painter paintersrp@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
