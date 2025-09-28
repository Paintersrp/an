package tasks

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"

	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/note"
	taskindex "github.com/Paintersrp/an/internal/services/tasks/index"
)

type Item struct {
	ID         int
	Content    string
	Completed  bool
	Path       string
	Line       int
	RelPath    string
	Due        *time.Time
	Scheduled  *time.Time
	Priority   string
	Owner      string
	Project    string
	References []string
}

type Index interface {
	AcquireSnapshot() (*taskindex.Snapshot, error)
}

type Service struct {
	handler  *handler.FileHandler
	index    Index
	openFunc func(string, bool) error
}

func NewService(h *handler.FileHandler, idx Index) *Service {
	return &Service{
		handler:  h,
		index:    idx,
		openFunc: note.OpenFromPath,
	}
}

func (s *Service) List() ([]Item, error) {
	if s == nil || s.handler == nil || s.index == nil {
		return nil, errors.New("task service is not configured")
	}

	snapshot, err := s.index.AcquireSnapshot()
	if err != nil {
		return nil, err
	}

	tasks := snapshot.Tasks()
	items := make([]Item, 0, len(tasks))
	vault := s.handler.VaultDir()

	for i, task := range tasks {
		rel := task.Path
		if relPath, err := filepath.Rel(vault, task.Path); err == nil {
			rel = filepath.ToSlash(relPath)
		}

		items = append(items, Item{
			ID:         i + 1,
			Content:    task.Content,
			Completed:  strings.EqualFold(task.Status, "checked"),
			Path:       task.Path,
			Line:       task.Line,
			RelPath:    rel,
			Due:        task.Metadata.DueDate,
			Scheduled:  task.Metadata.ScheduledDate,
			Priority:   task.Metadata.Priority,
			Owner:      task.Metadata.Owner,
			Project:    task.Metadata.Project,
			References: append([]string(nil), task.Metadata.References...),
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
		{Title: "Content", Width: 40},
		{Title: "Due", Width: 12},
		{Title: "Owner", Width: 16},
		{Title: "Priority", Width: 10},
		{Title: "Project", Width: 16},
		{Title: "Path", Width: 40},
	}

	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		status := "unchecked"
		if item.Completed {
			status = "checked"
		}
		due := ""
		if item.Due != nil {
			due = item.Due.Format("2006-01-02")
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", item.ID),
			status,
			item.Content,
			due,
			item.Owner,
			item.Priority,
			item.Project,
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
