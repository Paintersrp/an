package openVault

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/fs/zet"
	"github.com/Paintersrp/an/internal/config"
)

func NewCmdOpenVault(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vault",
		Aliases: []string{"v"},
		Short:   "Open the configured vault.",
		Long: heredoc.Doc(`
			Access your secure vault with this command. It opens the directory you've configured
			as your vault, allowing you to manage and interact with your zettels. Ensure your
			productivity by having all your notes organized and easily accessible.
		`),
		Example: heredoc.Doc(`
			# Opens the configured vault
			an open vault
      an o v
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}

	return cmd
}

func run() error {
	vaultDir := viper.GetString("vaultDir")

	if vaultDir == "" {
		fmt.Println("No vault directory configured.\nExiting")
		return errors.New("no vault directory")
	}
	zet.OpenFromPath(vaultDir)

	return nil
}
