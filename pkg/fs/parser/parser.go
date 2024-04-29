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

type Parser struct {
	DirPath     string
	TaskHandler *TaskHandler
	TagHandler  *TagHandler
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

// parse is a private method that reads and parses a Markdown file into an AST.
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

					if inTagsSection {
						// Handle tag parsing using TagHandler
						p.TagHandler.ParseTag(content)
					} else {
						// Handle task parsing using TaskHandler
						p.TaskHandler.ParseTask(content)
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
