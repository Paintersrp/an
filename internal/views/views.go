package views

import (
	"fmt"
	"strings"

	"github.com/Paintersrp/an/internal/handler"
)

var titlePrefixMap = map[string]string{
	"default":     "✅ - All",
	"orphan":      "❓ - Orphan",
	"unfulfilled": "⬜ - Unfulfilled",
	"archive":     "📦 - Archive",
	"trash":       "🗑️  - Trash", // Note the extra space before the dash
}

var sortFieldMap = map[int]string{
	0: "Title",
	1: "Subdirectory",
	2: "Modified",
}

func GetTitleForView(viewFlag string, sortField int, sortOrder int) string {
	prefix, ok := titlePrefixMap[viewFlag]
	if !ok {
		prefix = titlePrefixMap["default"]
	}

	sortFieldStr, ok := sortFieldMap[sortField]
	if !ok {
		sortFieldStr = "Unknown"
	}

	orderStr := "Ascending"
	if sortOrder == 1 {
		orderStr = "Descending"
	}

	return fmt.Sprintf(
		"%s View \nSort: %s (%s)",
		prefix,
		sortFieldStr,
		orderStr,
	)
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
	defaultExcludeDirs := []string{"archive", "trash"}
	defaultExcludeFiles := []string{}

	var (
		excludeDirs  []string
		excludeFiles []string
	)

	m, ok := vm.Views[viewFlag]
	if !ok {
		availableViews := vm.GetAvailableViews()
		return nil, fmt.Errorf(
			"invalid view: %s. Available views are: %s",
			viewFlag,
			availableViews,
		)
	}

	if len(m.ExcludeDirs) == 0 {
		excludeDirs = defaultExcludeDirs
	} else {
		excludeDirs = m.ExcludeDirs
	}

	if len(m.ExcludeFiles) == 0 {
		excludeFiles = defaultExcludeFiles
	} else {
		excludeFiles = m.ExcludeFiles
	}

	return vm.Handler.WalkFiles(excludeDirs, excludeFiles, viewFlag)
}
