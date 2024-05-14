package parser

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	tableTui "github.com/Paintersrp/an/internal/tui/table"
)

type Task struct {
	Status  string
	Content string
	ID      int
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

func (th *TaskHandler) ParseTask(content string) {
	if (strings.HasPrefix(content, "[ ]") || strings.HasPrefix(content, "[x]")) &&
		len(strings.TrimSpace(content[3:])) > 0 {
		status := "unchecked"
		if strings.HasPrefix(content, "[x]") {
			status = "checked"
		}
		th.AddTask(status, strings.TrimSpace(content[3:]))
	}
}

func (th *TaskHandler) AddTask(status, content string) {
	th.Tasks[th.NextID] = Task{
		ID:      th.NextID,
		Status:  status,
		Content: content,
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
		fmt.Printf(
			"ID: %d, Status: %s, Content: %s\n",
			task.ID,
			task.Status,
			task.Content,
		)
	}
}

func (th *TaskHandler) setupTasksTable() table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Status", Width: 10},
		{Title: "Content", Width: 100},
	}

	var rows []table.Row
	sorted := th.SortTasksByID("asc")
	for _, task := range sorted {
		rows = append(rows, []string{
			fmt.Sprintf("%d", task.ID),
			task.Status,
			task.Content,
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
