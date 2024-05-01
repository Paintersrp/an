package pin

import (
	"errors"
	"fmt"
	"os"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/fzf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin [query] --path {file_path}",
		Short: "Pin a file to be used with the echo command.",
		Long: `The pin command allows the user to specify a file that can be used with the echo command.
The path to the pinned file is saved in the configuration.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, c)
		},
	}

	cmd.Flags().
		StringP(
			"path",
			"p",
			"",
			"Manually enter the path to the file to pin without fuzzyfinding",
		)

	return cmd
}

func run(cmd *cobra.Command, args []string, c *config.Config) error {
	pathFlag, err := cmd.Flags().GetString("path")
	if err != nil {
		fmt.Printf("error retrieving path flag: %s\n", err)
		os.Exit(1)
	}

	if pathFlag == "" {
		vaultDir := viper.GetString("vaultDir")
		finder := fzf.NewFuzzyFinder(vaultDir, "Select file to pin.")

		if len(args) == 0 {
			choice, err := finder.Run(false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			c.ChangePin(choice, "text")
		} else {
			choice, err := finder.RunWithQuery(args[0], false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			c.ChangePin(choice, "text")
		}
	} else {
		if _, err := os.Stat(pathFlag); os.IsNotExist(err) {
			return errors.New("the specified file does not exist")
		}
		c.ChangePin(pathFlag, "text")
	}

	return nil
}
