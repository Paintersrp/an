package views

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Paintersrp/an/fs/handler"
)

type View struct {
	ExcludeDirs  []string
	ExcludeFiles []string
	OrphanOnly   bool
}

func GenerateViews(vaultDir string) map[string]View {
	views := make(map[string]View)

	views["default"] = View{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	views["archive"] = View{
		ExcludeDirs:  GetSubdirectories(vaultDir, "archive"),
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	views["orphan"] = View{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   true,
	}

	views["trash"] = View{
		ExcludeDirs:  GetSubdirectories(vaultDir, "trash"),
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	views["unfulfilled"] = View{
		ExcludeDirs:  []string{"archive", "trash"},
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	return views
}

func GetTitleForView(viewFlag string) string {
	var titlePrefix string
	switch viewFlag {
	case "archive":
		titlePrefix = "📦 - Archive"
	case "orphan":
		titlePrefix = "❓ - Orphan"
	case "trash":
		// second space is intentional
		titlePrefix = "🗑️  - Trash"
	case "unfulfilled":
		titlePrefix = "⬜ - Unfulfilled"
	default:
		titlePrefix = "✅ - Active"
	}

	return titlePrefix + " View"
}

func GetSubdirectories(directory, excludeDir string) []string {
	files, err := os.ReadDir(directory)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	var subDirs []string
	for _, f := range files {
		if f.IsDir() && f.Name() != excludeDir {

			subDir := strings.TrimPrefix(filepath.Join(directory, f.Name()), directory)
			subDir = strings.TrimPrefix(
				subDir,
				string(os.PathSeparator),
			)
			subDirs = append(subDirs, subDir)
		}
	}
	return subDirs
}

func GetAvailableViews(views map[string]View) string {
	var l []string
	for v := range views {
		l = append(l, v)
	}
	return strings.Join(l, ", ")
}

func GetFilesByView(
	views map[string]View,
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

	return handler.WalkFiles(vaultDir, excludeDirs, excludeFiles, viewFlag)
}

func getAvailableViews(views map[string]View) string {
	var l []string
	for v := range views {
		l = append(l, v)
	}
	return strings.Join(l, ", ")
}
