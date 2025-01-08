package views

import (
	"fmt"
	"strings"

	"github.com/Paintersrp/an/internal/handler"
	"github.com/charmbracelet/lipgloss"
)

var titlePrefixMap = map[string]string{
	"default":     "[1] All",
	"orphan":      "[2] Orphan",
	"unfulfilled": "[3] Unfulfilled",
	"archive":     "[4] Archive",
	"trash":       "[5] Trash",
}

var sortFieldMap = map[int]string{
	0: " [F1] Title",
	1: " [F2] Subdirectory",
	2: " [F3] Modified Date",
}

var sortOrderMap = map[int]string{
	0: "[F5] Ascending",
	1: "[F6] Descending",
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true)
	activeViewStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0AF")).
			Padding(0, 1)
	inactiveViewStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Padding(0, 1)
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			SetString("│")
	sortStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Bold(true)
)

func GetTitleForView(viewFlag string, sortField int, sortOrder int) string {
	// Handle view status
	views := []string{"default", "orphan", "unfulfilled", "archive", "trash"}
	var viewStatus []string
	for _, v := range views {
		prefix := titlePrefixMap[v]
		if v == viewFlag {
			viewStatus = append(viewStatus, activeViewStyle.Render(prefix))
		} else {
			viewStatus = append(viewStatus, inactiveViewStyle.Render(prefix))
		}
	}

	// Handle sort fields
	var sortStatus []string
	for i := 0; i < len(sortFieldMap); i++ {
		sortStr := sortFieldMap[i]
		if i == sortField {
			sortStatus = append(sortStatus, activeViewStyle.Render(sortStr))
		} else {
			sortStatus = append(sortStatus, inactiveViewStyle.Render(sortStr))
		}
	}

	// Handle sort order
	var orderStatus []string
	for i := 0; i < len(sortOrderMap); i++ {
		orderStr := sortOrderMap[i]
		if i == sortOrder {
			orderStatus = append(orderStatus, activeViewStyle.Render(orderStr))
		} else {
			orderStatus = append(orderStatus, inactiveViewStyle.Render(orderStr))
		}
	}

	viewLine := fmt.Sprintf("%s %s",
		titleStyle.Render("Views:"),
		strings.Join(viewStatus, dividerStyle.String()),
	)

	sortLine := fmt.Sprintf("%s %s %s %s",
		titleStyle.Render("Sort:"),
		strings.Join(sortStatus, dividerStyle.String()),
		dividerStyle.String(),
		strings.Join(orderStatus, dividerStyle.String()),
	)

	return fmt.Sprintf("%s\n%s", viewLine, sortLine)
}

// View represents a configuration for a specific view.
type View struct {
	ExcludeDirs  []string
	ExcludeFiles []string
	OrphanOnly   bool
}

// ViewManager manages available views and their configurations.
type ViewManager struct {
	Views   map[string]View
	Handler *handler.FileHandler
}

// NewViewManager creates a new ViewManager instance with default views.
func NewViewManager(h *handler.FileHandler, vaultDir string) *ViewManager {
	vm := &ViewManager{
		Views:   make(map[string]View),
		Handler: h,
	}

	vm.Views["default"] = View{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	vm.Views["orphan"] = View{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   true,
	}

	vm.Views["unfulfilled"] = View{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	vm.Views["archive"] = View{
		ExcludeDirs:  h.GetSubdirectories(vaultDir, "archive"),
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	vm.Views["trash"] = View{
		ExcludeDirs:  h.GetSubdirectories(vaultDir, "trash"),
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	return vm
}

func (vm *ViewManager) GetView(viewName string) (View, error) {
	view, ok := vm.Views[viewName]
	if !ok {
		return View{}, fmt.Errorf("invalid view: %s", viewName)
	}
	return view, nil
}

// GetAvailableViews returns a comma-separated list of available view names.
func (vm *ViewManager) GetAvailableViews() string {
	var viewNames []string
	for name := range vm.Views {
		viewNames = append(viewNames, name)
	}
	return strings.Join(viewNames, ", ")
}

func (vm *ViewManager) GetFilesByView(
	viewFlag string,
	vaultDir string,
) ([]string, error) {
	defaultExcludeDirs := []string{"archive"}
	defaultExcludeFiles := []string{}

	var (
		excludeDirs  []string
		excludeFiles []string
	)

	m, ok := vm.Views[viewFlag]
	if !ok {
		availableViews := vm.GetAvailableViews()
		panic(fmt.Errorf(
			"invalid view: %s. Available views are: %s",
			viewFlag,
			availableViews,
		))

	}
	if len(m.ExcludeDirs) == 0 {
		excludeDirs = defaultExcludeDirs
	} else {
		excludeDirs = m.ExcludeDirs
	}
	if len(excludeFiles) == 0 {
		excludeFiles = defaultExcludeFiles
	} else {
		excludeDirs = m.ExcludeFiles
	}

	return vm.Handler.WalkFiles(excludeDirs, excludeFiles, viewFlag)
}
