package register

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/auth/tui"
	"github.com/spf13/cobra"
)

func NewCmdRegister(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "register",
		Aliases: []string{"r"},
		Short:   "Register a new account",
		Long:    heredoc.Doc(``),
		Example: heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.Config.Token != "" {
				fmt.Println(
					"You are already authenticated. Please logout with the logout command if you'd like to change users.",
				)
			} else {
				if err := tui.Register(s); err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}
