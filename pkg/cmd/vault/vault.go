package vault

import (
	"fmt"
	"os"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdVault(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vault",
		Aliases: []string{"v"},
		Short:   "Open the configured vault.",
		Long:    ``,
		Example: "atomic vault",
		Run: func(cmd *cobra.Command, args []string) {
			vaultDir := viper.GetString("vaultDir")

			if vaultDir == "" {
				fmt.Println("No vault directory configured.\nExiting")
				os.Exit(1)
			}
			zet.OpenFromPath(vaultDir)

		},
	}

	return cmd
}
