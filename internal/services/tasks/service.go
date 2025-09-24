package tasks

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"

	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/parser"
)

type Item struct {
	ID        int
	Content   string
	Completed bool
	Path      string
	Line      int
	RelPath   string
}

type Service struct {
	handler       *handler.FileHandler
	parserFactory func(string) *parser.Parser
	openFunc      func(string, bool) error
}

func NewService(h *handler.FileHandler) *Service {
	return &Service{
		handler:       h,
		parserFactory: parser.NewParser,
		openFunc:      note.OpenFromPath,
	}
}

func (s *Service) List() ([]Item, error) {
	if s == nil || s.handler == nil {
		return nil, errors.New("task service is not configured")
	}

	vault := s.handler.VaultDir()
	p := s.parserFactory(vault)
	if err := p.Walk(); err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(p.TaskHandler.Tasks))
	for _, task := range p.TaskHandler.Tasks {
		rel := task.Path
		if relPath, err := filepath.Rel(vault, task.Path); err == nil {
			rel = filepath.ToSlash(relPath)
		}

		items = append(items, Item{
			ID:        task.ID,
			Content:   task.Content,
			Completed: strings.EqualFold(task.Status, "checked"),
			Path:      task.Path,
			Line:      task.Line,
			RelPath:   rel,
		})
	}

	return items, nil
}

func (s *Service) Toggle(path string, line int) (bool, error) {
	if s == nil || s.handler == nil {
		return false, errors.New("task service is not configured")
	}

	data, err := s.handler.ReadFile(path)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(data), "\n")
	if line <= 0 || line > len(lines) {
		return false, fmt.Errorf("line %d out of range", line)
	}

	target := lines[line-1]
	switch {
	case strings.Contains(target, "[ ]"):
		idx := strings.Index(target, "[ ]")
		lines[line-1] = target[:idx] + strings.Replace(target[idx:], "[ ]", "[x]", 1)
		if err := s.handler.WriteFile(path, []byte(strings.Join(lines, "\n"))); err != nil {
			return false, err
		}
		return true, nil
	case strings.Contains(target, "[x]"):
		idx := strings.Index(target, "[x]")
		lines[line-1] = target[:idx] + strings.Replace(target[idx:], "[x]", "[ ]", 1)
		if err := s.handler.WriteFile(path, []byte(strings.Join(lines, "\n"))); err != nil {
			return false, err
		}
		return false, nil
	default:
		return false, fmt.Errorf("no markdown task found on line %d", line)
	}
}

func (s *Service) Open(path string) error {
	if s == nil {
		return errors.New("task service is not configured")
	}
	return s.openFunc(path, false)
}

func TableFromItems(items []Item, height int) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Status", Width: 10},
		{Title: "Content", Width: 60},
		{Title: "Path", Width: 40},
	}

	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		status := "unchecked"
		if item.Completed {
			status = "checked"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", item.ID),
			status,
			item.Content,
			item.RelPath,
		})
	}

	return table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
}
