package login

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/auth/tui"
	"github.com/spf13/cobra"
)

func NewCmdLogin(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"l"},
		Short:   "Log in to your account",
		Long: heredoc.Doc(`
			Log in to your account with your email and password.
			Upon successful login, your authentication token will be stored in a ~/.keys directory.
		`),
		Example: heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.Config.Token != "" {
				fmt.Println(
					"You are already authenticated. Please logout with the logout command if you'd like to change users.",
				)
			} else {
				if err := tui.Login(s); err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}
