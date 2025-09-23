package pinAdd

import (
	"errors"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/fzf"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/shared/flags"
)

func NewCmdPinAdd(s *state.State, pinType string) *cobra.Command {
	var check bool

	cmd := &cobra.Command{
		Use:     "add [query] [--name pin_name] [--path file_path] [--check]",
		Aliases: []string{"a"},
		Short:   "Pin a file to be used with the echo command or check the current pin.",
		Long: heredoc.Doc(`
			The add pin command allows you to specify a file that can be used with the echo command,
			or check the currently pinned file. The path to the pinned file is saved in the configuration.

			Examples:
			  an pin add --path /path/to/myfile.txt
			  an pin add my-note
			  an pin add --check
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if check {
				ws := s.Config.MustWorkspace()
				if pinType == "task" {
					fmt.Println("Current pinned file:", ws.PinnedTaskFile)
				} else {
					fmt.Println("Current pinned file:", ws.PinnedFile)
				}
				return nil
			}
			return run(cmd, args, s, pinType)
		},
	}

	cmd.Flags().BoolVarP(&check, "check", "c", false, "Check the current pinned file")
	flags.AddName(cmd, "Name for new pin.")
	flags.AddPath(cmd)

	return cmd
}

func run(
	cmd *cobra.Command,
	args []string,
	s *state.State,
	pinType string,
) error {
	path := flags.HandlePath(cmd)
	name, err := flags.HandleName(cmd)
	if err != nil {
		return err
	}

	if path == "" {
		vaultDir := viper.GetString("vaultDir")
		finder := fzf.NewFuzzyFinder(vaultDir, "Select file to pin.")

		if len(args) == 0 {
			choice, err := finder.Run(false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			if err := s.Config.ChangePin(choice, pinType, name); err != nil {
				return err
			}
		} else {
			choice, err := finder.RunWithQuery(args[0], false)
			if err != nil {
				fmt.Printf("error fuzzyfinding note: %s", err)
			}

			if err := s.Config.ChangePin(choice, pinType, name); err != nil {
				return err
			}
		}
	} else {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New("the specified file does not exist")
		}
		if err := s.Config.ChangePin(path, pinType, name); err != nil {
			return err
		}
	}

	return nil
}
