package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type TaskFile struct {
	Path      string
	TaskCount int
}

func ParseMarkdownFile(
	path string,
	taskMap *map[string][]string,
) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	parser := goldmark.DefaultParser()
	reader := text.NewReader(source)
	document := parser.Parse(reader)

	var tasks []string

	ast.Walk(
		document,
		func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				if listItem, ok := n.(*ast.ListItem); ok {
					content := strings.TrimSpace(
						string(listItem.Text(source)),
					)
					if (strings.HasPrefix(content, "[ ]") || strings.HasPrefix(content, "[x]")) &&
						len(strings.TrimSpace(content[3:])) > 0 {
						tasks = append(tasks, content)
					}
				}
			}
			return ast.WalkContinue, nil
		},
	)

	(*taskMap)[path] = tasks

	return nil
}

func WalkDir(dirPath string, taskMap *map[string][]string) error {
	return filepath.Walk(
		dirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf(
					"error walking the path %q: %w",
					path,
					err,
				)
			}
			if !info.IsDir() && filepath.Ext(path) == ".md" {
				if err := ParseMarkdownFile(path, taskMap); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
