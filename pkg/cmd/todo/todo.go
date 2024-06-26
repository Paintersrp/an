package todo

// full wip

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
)

func NewCmdTodo(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "todo",
		Aliases: []string{"td"},
		Short:   "Parse TODO comments and generate a markdown checklist",
		Long:    heredoc.Doc(``),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}

	return cmd
}

func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return err
	}

	todoRegex := regexp.MustCompile(
		`(?i)(?:^|\s)(?:\/\/|#|\/\*|\*|--|<!--)\s*TODO:\s*(.+?)(?:\*\/|-->|$)`,
	)

	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return err
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("Error reading file:", path, err)
				return err
			}

			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if strings.Contains(line, "- [ ]") {
					continue
				}

				matches := todoRegex.FindStringSubmatch(line)
				if len(matches) > 1 {
					fmt.Printf("\n- [ ] %s\n", strings.TrimSpace(matches[1]))
					fmt.Printf("    - File: %s\n", path)
					fmt.Printf("    - Line: %d\n", i+1)
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking through files:", err)
		return err
	}
	return nil
}
