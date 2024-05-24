package auth

import (
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/auth/check"
	"github.com/Paintersrp/an/pkg/cmd/auth/login"
	"github.com/Paintersrp/an/pkg/cmd/auth/logout"
	"github.com/Paintersrp/an/pkg/cmd/auth/note"
	"github.com/Paintersrp/an/pkg/cmd/auth/register"
)

func NewCmdAuth(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Aliases: []string{"a"},
		Short:   "Authenticate to the application.",
	}

	cmd.AddCommand(register.NewCmdRegister(s))
	cmd.AddCommand(login.NewCmdLogin(s))
	cmd.AddCommand(logout.NewCmdLogout(s))
	cmd.AddCommand(check.NewCmdCheck(s))

	cmd.AddCommand(note.NewCmdNoteTest(s))

	return cmd
}
