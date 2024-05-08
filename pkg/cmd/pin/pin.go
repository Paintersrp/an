package pin

import (
	"errors"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/flags"
	"github.com/Paintersrp/an/pkg/fs/fzf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdPin(c *config.Config) *cobra.Command {
	var check bool
	var name string

	cmd := &cobra.Command{
		Use:     "pin [query] [--name pin_name] [--path file_path] [--check]",
		Aliases: []string{"p"},
		Short:   "Pin a file to be used with the echo command or check the current pin.",
		Long: heredoc.Doc(`
			The pin command allows you to specify a file that can be used with the echo command,
			or check the currently pinned file. The path to the pinned file is saved in the configuration.

			Examples:
			  an pin --path /path/to/myfile.txt
			  an pin my-note
			  an pin --check
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if check {
				fmt.Println("Current pinned file:", c.PinnedFile)
				return nil
			}
			return run(cmd, args, c, name)
		},
	}

	cmd.Flags().BoolVarP(&check, "check", "c", false, "Check the current pinned file")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Save as a named pin")
	flags.AddPath(cmd)

	return cmd
}

func run(cmd *cobra.Command, args []string, c *config.Config, name string) error {
	path := flags.HandlePath(cmd)

	if path == "" {
		vaultDir := viper.GetString("vaultDir")
		finder := fzf.NewFuzzyFinder(vaultDir, "Select file to pin.")

		if len(args) == 0 {
			choice, err := finder.Run(false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			c.ChangePin(choice, "text", name)
		} else {
			choice, err := finder.RunWithQuery(args[0], false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			c.ChangePin(choice, "text", name)
		}
	} else {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New("the specified file does not exist")
		}
		c.ChangePin(path, "text", name)
	}

	return nil
}
