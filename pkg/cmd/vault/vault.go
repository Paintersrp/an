package vault

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/vault/vaultAdd"
	"github.com/Paintersrp/an/pkg/cmd/vault/vaultChange"
	"github.com/Paintersrp/an/pkg/cmd/vault/vaultInit"
	"github.com/Paintersrp/an/pkg/cmd/vault/vaultSync"
	"github.com/spf13/cobra"
)

func NewCmdVault(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vault [name]",
		Aliases: []string{"v"},
		Short:   "",
		Long:    heredoc.Doc(`\`),
		Args:    cobra.ExactArgs(1),
	}

	cmd.AddCommand(vaultAdd.NewCmdVaultAdd(s))
	cmd.AddCommand(vaultChange.NewCmdVaultChange(s))
	cmd.AddCommand(vaultInit.NewCmdVaultInit(s))
	cmd.AddCommand(vaultSync.NewCmdVaultSync(s))

	return cmd
}
