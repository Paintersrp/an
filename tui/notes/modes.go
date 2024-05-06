package notes

import (
	"fmt"
	"strings"

	"github.com/Paintersrp/an/pkg/fs/fzf"
)

type ModeConfig struct {
	ExcludeDirs  []string
	ExcludeFiles []string
	OrphanOnly   bool
}

func GenerateModes(vaultDir string) map[string]ModeConfig {
	modes := make(map[string]ModeConfig)

	// Default mode configuration
	modes["default"] = ModeConfig{
		ExcludeDirs:  []string{"archive"},
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	// Generate archive mode configuration
	modes["archive"] = ModeConfig{
		ExcludeDirs:  getSubdirectories(vaultDir, "archive"),
		ExcludeFiles: []string{},
		OrphanOnly:   false,
	}

	modes["orphan"] = ModeConfig{
		ExcludeDirs:  []string{"archive"},
		ExcludeFiles: []string{},
		OrphanOnly:   true,
	}

	// Add more modes as needed

	return modes
}

func getTitleForMode(modeFlag string) string {
	var title string
	switch modeFlag {
	case "archive":
		title = "Archived Notes"
	case "default":
		title = "Active Notes"
	case "orphan":
		title = "Orphaned Notes"
	default:
		title = "Active Notes"
	}

	return title
}

func getFilesByMode(
	modes map[string]ModeConfig,
	modeFlag string,
	vaultDir string,
) ([]string, error) {
	defaultExcludeDirs := []string{"archive"}
	defaultExcludeFiles := []string{}

	var (
		excludeDirs  []string
		excludeFiles []string
	)

	modeConfig, ok := modes[modeFlag]
	if !ok {
		availableModes := getAvailableModes(modes)
		panic(fmt.Errorf(
			"invalid mode: %s. Available modes are: %s",
			modeFlag,
			availableModes,
		))

	}
	// Use the provided arguments if they are not empty; otherwise, use the defaults
	if len(modeConfig.ExcludeDirs) == 0 {
		excludeDirs = defaultExcludeDirs
	} else {
		excludeDirs = modeConfig.ExcludeDirs
	}
	if len(excludeFiles) == 0 {
		excludeFiles = defaultExcludeFiles
	} else {
		excludeDirs = modeConfig.ExcludeFiles
	}

	return fzf.StaticListFiles(vaultDir, excludeDirs, excludeFiles, modeFlag)
}

func getAvailableModes(modes map[string]ModeConfig) string {
	var modesList []string
	for mode := range modes {
		modesList = append(modesList, mode)
	}
	return strings.Join(modesList, ", ")
}
