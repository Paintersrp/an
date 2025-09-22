package pathutil

import (
	"path/filepath"
	"strings"
)

// NormalizePath converts Windows-style separators to the current platform's separator
// and cleans the resulting path.
func NormalizePath(p string) string {
	if p == "" {
		return ""
	}

	// Replace Windows separators and collapse redundant separators/segments.
	replaced := strings.ReplaceAll(p, "\\", "/")
	return filepath.Clean(filepath.FromSlash(replaced))
}

// VaultRelative returns the path to target relative to the provided vault directory.
// The returned path always uses forward slashes to simplify downstream processing
// and ensure platform agnosticism.
func VaultRelative(vaultDir, target string) (string, error) {
	base := NormalizePath(vaultDir)
	cleanedTarget := NormalizePath(target)

	rel, err := filepath.Rel(base, cleanedTarget)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(rel), nil
}

// VaultRelativeComponents splits the relative path from VaultRelative into the first
// directory (if present) and the remaining path.
func VaultRelativeComponents(vaultDir, target string) (string, string, error) {
	rel, err := VaultRelative(vaultDir, target)
	if err != nil {
		return "", "", err
	}

	rel = strings.TrimPrefix(rel, "./")
	if rel == "." || rel == "" {
		return "", "", nil
	}

	parts := strings.Split(rel, "/")
	if len(parts) == 1 {
		return "", parts[0], nil
	}

	return parts[0], strings.Join(parts[1:], "/"), nil
}
