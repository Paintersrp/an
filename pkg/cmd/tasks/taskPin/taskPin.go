package taskPin

import (
	"errors"
	"fmt"
	"os"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/fzf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdTaskPin(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pin [query] --path {file_path}",
		Aliases: []string{"p"},
		Short:   "Pin a task file",
		Long:    `The pin command is used to pin a task file, making it the target for other task operations.`,
		Example: `
    # Pin a task file
    an-cli tasks pin -p ~/tasks.md
    an-cli tasks pin
    an-cli tasks pin "texas toast"
    `,
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
		finder := fzf.NewFuzzyFinder(vaultDir, "Select file to pin")

		if len(args) == 0 {
			choice, err := finder.Run(false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			c.ChangePin(choice, "task")
		} else {
			choice, err := finder.RunWithQuery(args[0], false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			c.ChangePin(choice, "task")
		}
	} else {
		if _, err := os.Stat(pathFlag); os.IsNotExist(err) {
			return errors.New("the specified task echo file does not exist")
		}
		c.ChangePin(pathFlag, "task")
	}

	return nil
}
