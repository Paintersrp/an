package addSubdir

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/cobra"
)

func isValidDirName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("directory name cannot be empty")
	}
	if name != filepath.Clean(name) {
		return errors.New("directory name contains invalid characters or sequence")
	}
	if strings.Contains(name, string(filepath.Separator)) {
		return errors.New("directory name must not contain path separators")
	}
	return nil
}

func NewCmdAddSubdir(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-subdir [name]",
		Aliases: []string{"as", "add-s", "a-s"},
		Short:   "Add a subdirectory to the list of available directories",
		Long: heredoc.Doc(`
			This command adds a subdirectory to the list of available projects in the global persistent configuration.
			It's a quick way to expand your project's organization by categorizing related files under a common directory.

			Examples:
			# Adds a new subdirectory named 'Go-CLI' to the list of directories
			an add-subdir Go-CLI
		`),
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if err := isValidDirName(name); err != nil {
				cmd.Println("Error:", err)
				return
			}
			c.AddSubdir(name)
		},
	}

	return cmd
}
