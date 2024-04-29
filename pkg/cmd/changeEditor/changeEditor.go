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
package changeEditor

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdChangeEditor(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change-editor [editor]",
		Short: "Change the default text editor",
		Long: `The change-editor command updates the default text editor used by the application and saves the new setting to the configuration file.
This allows for customization of the text editing experience to suit the user's preferences.`,
		Example: `
    # Change the default editor to 'vim'
    an-cli change-editor vim
    `,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c.ChangeEditor(args[0])
		},
	}

	return cmd
}
