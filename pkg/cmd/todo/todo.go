package todo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
)

type options struct {
	capture     bool
	destination string
}

type todoEntry struct {
	Text    string
	Path    string
	RelPath string
	Line    int
}

func NewCmdTodo(c *config.Config) *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:     "todo",
		Aliases: []string{"td"},
		Short:   "Scan for TODO comments and surface them as actionable tasks",
		Long: heredoc.Doc(`
                        The todo command walks the current working directory, extracts TODO comments, and
                        converts them into Markdown tasks. Use --capture to append the results to your vault's
                        pinned task note (or a destination of your choice) so they appear in the Tasks TUI.
                `),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(c, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.capture, "capture", false, "append captured TODOs to the vault inbox")
	cmd.Flags().StringVar(&opts.destination, "dest", "", "override the destination note (relative to the vault)")

	return cmd
}

func run(cfg *config.Config, opts options) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine working directory: %w", err)
	}

	entries, err := collectTODOs(cwd)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("no TODO comments found")
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RelPath == entries[j].RelPath {
			return entries[i].Line < entries[j].Line
		}
		return entries[i].RelPath < entries[j].RelPath
	})

	if opts.capture {
		if cfg == nil {
			return errors.New("configuration is required to capture TODOs")
		}
		if err := appendToVault(cfg, entries, opts.destination); err != nil {
			return err
		}
		fmt.Printf("captured %d TODO items into your vault\n", len(entries))
		return nil
	}

	for _, entry := range entries {
		fmt.Printf("- [ ] %s\n", entry.Text)
		fmt.Printf("    - File: %s\n", entry.RelPath)
		fmt.Printf("    - Line: %d\n", entry.Line)
	}

	return nil
}

func collectTODOs(root string) ([]todoEntry, error) {
	todoRegex := regexp.MustCompile(
		`(?i)(?:^|\s)(?:\/\/|#|\/\*|\*|--|<!--)\s*TODO:\s*(.+?)(?:\*\/|-->|$)`,
	)

	entries := make([]todoEntry, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.Contains(line, "- [ ]") {
				continue
			}

			matches := todoRegex.FindStringSubmatch(line)
			if len(matches) < 2 {
				continue
			}
			text := strings.TrimSpace(matches[1])
			if text == "" {
				continue
			}

			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}

			entries = append(entries, todoEntry{
				Text:    text,
				Path:    path,
				RelPath: filepath.ToSlash(rel),
				Line:    i + 1,
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking through files: %w", err)
	}

	return entries, nil
}

func appendToVault(cfg *config.Config, entries []todoEntry, override string) error {
	ws := cfg.MustWorkspace()
	if ws == nil {
		return errors.New("active workspace is not configured")
	}

	dest, err := resolveDestination(ws, override)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("failed to prepare inbox location: %w", err)
	}

	file, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open inbox note: %w", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "\n## TODO Capture %s\n\n", time.Now().Format(time.RFC3339))
	for _, entry := range entries {
		fmt.Fprintf(file, "- [ ] %s @project(codebase) (%s:%d)\n", entry.Text, entry.RelPath, entry.Line)
	}

	return nil
}

func resolveDestination(ws *config.Workspace, override string) (string, error) {
	base := ws.VaultDir
	if base == "" {
		return "", errors.New("workspace vault directory is not configured")
	}

	chosen := strings.TrimSpace(override)
	if chosen == "" {
		chosen = ws.PinnedTaskFile
	}
	if chosen == "" {
		chosen = filepath.Join("inbox", "code-todos.md")
	}

	if !filepath.IsAbs(chosen) {
		chosen = filepath.Join(base, chosen)
	}

	cleaned := filepath.Clean(chosen)
	rel, err := filepath.Rel(base, cleaned)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("destination %q is outside the vault", cleaned)
	}

	return cleaned, nil
}
