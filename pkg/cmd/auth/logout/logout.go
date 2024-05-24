package logout

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/spf13/cobra"
)

func NewCmdLogout(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "Logout of your account",
		Long:    heredoc.Doc(``),
		Example: heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			s.Config.ChangeToken("")
			fmt.Println("Successfully logged out.")
			return nil
		},
	}

	return cmd
}
