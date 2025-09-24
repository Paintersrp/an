package parser

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	tableTui "github.com/Paintersrp/an/internal/tui/table"
)

type Task struct {
	Status   string
	Content  string
	ID       int
	Path     string
	Line     int
	Metadata TaskMetadata
}

type TaskMetadata struct {
	DueDate       *time.Time
	ScheduledDate *time.Time
	Priority      string
	Owner         string
	Project       string
	References    []string
	RawTokens     map[string]string
}

type TaskHandler struct {
	Tasks  map[int]Task
	NextID int
}

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{
		Tasks:  make(map[int]Task),
		NextID: 1,
	}
}

func (th *TaskHandler) ParseTask(content, path string, line int) {
	if (strings.HasPrefix(content, "[ ]") || strings.HasPrefix(content, "[x]")) &&
		len(strings.TrimSpace(content[3:])) > 0 {
		status := "unchecked"
		if strings.HasPrefix(content, "[x]") {
			status = "checked"
		}

		body := strings.TrimSpace(content[3:])
		cleaned, metadata := ExtractTaskMetadata(body)
		if cleaned == "" {
			return
		}

		lowered := strings.ToLower(cleaned)
		if strings.HasPrefix(lowered, "tags:") {
			return
		}

		th.AddTask(status, cleaned, path, line, metadata)
	}
}

func (th *TaskHandler) AddTask(status, content, path string, line int, metadata TaskMetadata) {
	th.Tasks[th.NextID] = Task{
		ID:       th.NextID,
		Status:   status,
		Content:  content,
		Path:     path,
		Line:     line,
		Metadata: metadata,
	}
	th.NextID++
}

func (th *TaskHandler) SortTasksByID(order string) []Task {
	var tasks []Task
	for _, task := range th.Tasks {
		tasks = append(tasks, task)
	}

	switch order {
	case "asc":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].ID < tasks[j].ID
		})
	case "desc":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].ID > tasks[j].ID
		})
	default:
		fmt.Println(
			"Invalid sort order. Use 'asc' for ascending or 'desc' for descending.",
		)
		return nil
	}

	return tasks
}

// sortTasksByStatus is a private method that sorts tasks by status.
func (th *TaskHandler) SortTasksByStatus(order string) []Task {
	var tasks []Task
	for _, task := range th.Tasks {
		tasks = append(tasks, task)
	}

	switch order {
	case "asc":
		sort.SliceStable(tasks, func(i, j int) bool {
			if tasks[i].Status == tasks[j].Status {
				return tasks[i].ID < tasks[j].ID
			}
			return tasks[i].Status < tasks[j].Status
		})
	case "desc":
		sort.SliceStable(tasks, func(i, j int) bool {
			if tasks[i].Status == tasks[j].Status {
				return tasks[i].ID < tasks[j].ID
			}
			return tasks[i].Status > tasks[j].Status
		})
	default:
		fmt.Println(
			"Invalid sort order. Use 'asc' for ascending or 'desc' for descending.",
		)
		return nil
	}

	return tasks
}

// PrintTasks is a public method that prints tasks sorted by the specified type and order.
func (th *TaskHandler) PrintTasks(sortType, sortOrder string) {
	var sortedTasks []Task
	switch sortType {
	case "id":
		sortedTasks = th.SortTasksByID(sortOrder)
	case "status":
		sortedTasks = th.SortTasksByStatus(sortOrder)
	default:
		fmt.Println("Invalid sort type. Use 'id' or 'status'.")
		return
	}

	if sortedTasks != nil {
		fmt.Println("\nSorted Tasks:")
		th.printTasks(sortedTasks)
	}
}

func (th *TaskHandler) printTasks(tasks []Task) {
	for _, task := range tasks {
		meta := task.Metadata
		var details []string
		if meta.DueDate != nil {
			details = append(details, fmt.Sprintf("due %s", meta.DueDate.Format("2006-01-02")))
		}
		if meta.ScheduledDate != nil {
			details = append(details, fmt.Sprintf("scheduled %s", meta.ScheduledDate.Format("2006-01-02")))
		}
		if meta.Priority != "" {
			details = append(details, fmt.Sprintf("priority %s", meta.Priority))
		}
		if meta.Owner != "" {
			details = append(details, fmt.Sprintf("owner %s", meta.Owner))
		}
		if meta.Project != "" {
			details = append(details, fmt.Sprintf("project %s", meta.Project))
		}

		line := fmt.Sprintf("ID: %d, Status: %s, Content: %s", task.ID, task.Status, task.Content)
		if len(details) > 0 {
			line = fmt.Sprintf("%s (%s)", line, strings.Join(details, ", "))
		}
		fmt.Println(line)

		if len(meta.References) > 0 {
			fmt.Printf("  refs: %s\n", strings.Join(meta.References, ", "))
		}
	}
}

func (th *TaskHandler) setupTasksTable() table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Status", Width: 10},
		{Title: "Content", Width: 60},
		{Title: "Due", Width: 12},
		{Title: "Owner", Width: 18},
		{Title: "Priority", Width: 10},
		{Title: "Project", Width: 18},
	}

	var rows []table.Row
	sorted := th.SortTasksByID("asc")
	for _, task := range sorted {
		due := ""
		if task.Metadata.DueDate != nil {
			due = task.Metadata.DueDate.Format("2006-01-02")
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", task.ID),
			task.Status,
			task.Content,
			due,
			task.Metadata.Owner,
			task.Metadata.Priority,
			task.Metadata.Project,
		})
	}

	tableCfg := tableTui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Focused: true,
		Height:  20,
	}

	t := tableCfg.ReturnTable()
	return t
}

func (th *TaskHandler) ShowTasksTable() {
	t := th.setupTasksTable()
	m := tableTui.NewTableModel(t)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
