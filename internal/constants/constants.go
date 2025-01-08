package constants

const (
	ConfigFile     = `cfg`
	ConfigFileType = `yaml`
	ConfigDir      = `/.an/`
)

// AvailableEditors is a slice of supported text editors
var AvailableEditors = []string{
	"nvim",
	"vim",
	"nano",
	"emacs",
	"micro",
	"kak",
	"helix",
}

// AvailableEditors is a slice of supported file saving modes
var AvailableModes = []string{
	"strict",
	"confirm",
	"free",
}

// ValidEditors is a map for quick lookup of supported editors
var ValidEditors = map[string]bool{
	"nvim":  true, // Neovim - The Messiah
	"vim":   true,
	"nano":  true,
	"emacs": true,
	"micro": true,
	"kak":   true,
	"helix": true,
}

// ValidModes is a map for quick lookup of supported file saving modes
var ValidModes = map[string]bool{
	"strict":  true,
	"confirm": true,
	"free":    true,
}
