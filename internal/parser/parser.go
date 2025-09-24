package parser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type Parser struct {
	TaskHandler *TaskHandler
	TagHandler  *TagHandler
	DirPath     string
}

func NewParser(dirPath string) *Parser {
	return &Parser{
		DirPath:     dirPath,
		TaskHandler: NewTaskHandler(),
		TagHandler:  NewTagHandler(),
	}
}

// Walk traverses the directory set in the Parser and processes Markdown files.
func (p *Parser) Walk() error {
	return filepath.Walk(
		p.DirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf(
					"error walking the path %q: %w",
					path,
					err,
				)
			}
			if !info.IsDir() && filepath.Ext(path) == ".md" {
				if err := p.parse(path); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func (p *Parser) parse(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	parser := goldmark.DefaultParser()
	reader := text.NewReader(source)
	document := parser.Parse(reader)

	var inTagsSection bool

	ast.Walk(
		document,
		func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				switch n := n.(type) {
				case *ast.ListItem:
					content := strings.TrimSpace(string(n.Text(source)))
					line := 0
					if lines := n.Lines(); lines != nil && lines.Len() > 0 {
						segment := lines.At(0)
						line = 1 + bytes.Count(source[:segment.Start], []byte("\n"))
					} else if child := n.FirstChild(); child != nil {
						if clines := child.Lines(); clines != nil && clines.Len() > 0 {
							segment := clines.At(0)
							line = 1 + bytes.Count(source[:segment.Start], []byte("\n"))
						}
					}

					if inTagsSection {
						p.TagHandler.ParseTag(content)
					} else {
						p.TaskHandler.ParseTask(content, path, line)
					}
				case *ast.Text:
					content := strings.TrimSpace(string(n.Text(source)))

					if content == "tags:" {
						inTagsSection = true
					}
				}
			} else {
				if _, ok := n.(*ast.List); ok && inTagsSection {
					inTagsSection = false
				}
			}
			return ast.WalkContinue, nil
		},
	)

	return nil
}

func (p *Parser) PrintTagCounts() {
	p.TagHandler.PrintTagCounts()
}

func (p *Parser) PrintSortedTagCounts(order string) {
	p.TagHandler.PrintSortedTagCounts(order)
}

func (p *Parser) ShowTagTable() {
	p.TagHandler.ShowTagTable()
}

func (p *Parser) PrintTasks(sortType, sortOrder string) {
	p.TaskHandler.PrintTasks(sortType, sortOrder)
}

func (p *Parser) ShowTasksTable() {
	p.TaskHandler.ShowTasksTable()
}
