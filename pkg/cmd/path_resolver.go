package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Paintersrp/an/internal/state"
	"github.com/spf13/cobra"
)

func ResolveVaultPath(cmd *cobra.Command, s *state.State, arg string) (string, error) {
	if s == nil || s.Config == nil {
		return "", fmt.Errorf("state configuration is not initialized")
	}
	vaultDir := filepath.Clean(s.Config.MustWorkspace().VaultDir)
	if vaultDir == "" {
		return "", fmt.Errorf("vault directory is not configured")
	}
	if arg == "" {
		return "", fmt.Errorf("a path argument is required")
	}

	var resolved string
	if filepath.IsAbs(arg) {
		resolved = filepath.Clean(arg)
	} else {
		resolved = resolveRelative(cmd, vaultDir, arg)
	}

	if err := ensureWithinVault(vaultDir, resolved); err != nil {
		return "", err
	}

	return resolved, nil
}

func resolveRelative(cmd *cobra.Command, vaultDir, arg string) string {
	relPath := filepath.Clean(arg)
	if relPath == "." {
		relPath = ""
	}

	targetDir := inferTargetDir(cmd)
	if targetDir == "" {
		if relPath == "" {
			return vaultDir
		}
		return filepath.Join(vaultDir, relPath)
	}

	firstSegment := relPath
	if idx := strings.Index(relPath, string(filepath.Separator)); idx != -1 {
		firstSegment = relPath[:idx]
	}

	if firstSegment == targetDir {
		return filepath.Join(vaultDir, relPath)
	}

	return filepath.Join(vaultDir, targetDir, relPath)
}

func inferTargetDir(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}

	switch cmd.Name() {
	case "unarchive":
		return "archive"
	case "untrash":
		return "trash"
	default:
		return ""
	}
}

func ensureWithinVault(vaultDir, resolved string) error {
	rel, err := filepath.Rel(vaultDir, resolved)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q relative to vault %q: %w", resolved, vaultDir, err)
	}

	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside the vault %q", resolved, vaultDir)
	}

	return nil
}
