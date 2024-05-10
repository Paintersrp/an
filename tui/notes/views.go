package notes

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

	views["unfulfilled"] = ViewConfig{
		ExcludeDirs:  []string{"archive", "trash"},
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
	case "unfulfilled":
		titlePrefix = "⬜ - Unfulfilled"
	default:
		titlePrefix = "✅ - Active"
	}

	return titlePrefix + " View"
}
