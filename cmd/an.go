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
package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/tui/initialize"
	"github.com/Paintersrp/an/pkg/cmd/root"
)

func Execute() {
	s, err := state.NewState()

	if err != nil {
		var initErr *config.ConfigInitError
		if errors.As(err, &initErr) {
			err := initialize.Run()
			cobra.CheckErr(err)

			s, err := state.NewState()
			cobra.CheckErr(err) // TODO: or loop again if failed?

			executeRoot(s)
		} else {
			cobra.CheckErr(err)
		}
	} else {
		executeRoot(s)
	}
}

func executeRoot(s *state.State) {
	cmd, err := root.NewCmdRoot(s)
	cobra.CheckErr(err)

	execErr := cmd.Execute()
	cobra.CheckErr(execErr)
}
