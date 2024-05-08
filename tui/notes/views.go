package notes

import (
	"fmt"
	"strings"

	"github.com/Paintersrp/an/pkg/fs/fzf"
)

type ViewConfig struct {
	ExcludeDirs  []string
	ExcludeFiles []string
	OrphanOnly   bool
}

func GenerateViews(vaultDir string) map[string]ViewConfig {
	views := make(map[string]ViewConfig)

	views["default"] = ViewConfig{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	views["archive"] = ViewConfig{
		ExcludeDirs:  getSubdirectories(vaultDir, "archive"),
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	views["orphan"] = ViewConfig{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   true,
	}

	views["trash"] = ViewConfig{
		ExcludeDirs:  getSubdirectories(vaultDir, "trash"),
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	return views
}

func getTitleForView(viewFlag string) string {
	var titlePrefix string
	switch viewFlag {
	case "archive":
		titlePrefix = "📦 - Archive"
	case "orphan":
		titlePrefix = "❓ - Orphan"
	case "trash":
		// second space is intentional
		titlePrefix = "🗑️  - Trash"
	default:
		titlePrefix = "✅ - Active"
	}

	return titlePrefix + " View"
}

func getFilesByView(
	views map[string]ViewConfig,
	viewFlag string,
	vaultDir string,
) ([]string, error) {
	defaultExcludeDirs := []string{"archive"}
	defaultExcludeFiles := []string{}

	var (
		excludeDirs  []string
		excludeFiles []string
	)

	m, ok := views[viewFlag]
	if !ok {
		availableViews := getAvailableViews(views)
		panic(fmt.Errorf(
			"invalid view: %s. Available views are: %s",
			viewFlag,
			availableViews,
		))

	}
	// Use the provided arguments if they are not empty; otherwise, use the defaults
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

	return fzf.StaticListFiles(vaultDir, excludeDirs, excludeFiles, viewFlag)
}

func getAvailableViews(views map[string]ViewConfig) string {
	var l []string
	for v := range views {
		l = append(l, v)
	}
	return strings.Join(l, ", ")
}
