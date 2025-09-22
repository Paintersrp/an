package pathutil

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultRelativeReturnsForwardSlashes(t *testing.T) {
	vaultParts := []string{"home", "user", "vault"}
	fileParts := append(append([]string{}, vaultParts...), "subdir", "file.md")

	posixVault := filepath.Join(vaultParts...)
	posixFile := filepath.Join(fileParts...)

	rel, err := VaultRelative(posixVault, posixFile)
	if err != nil {
		t.Fatalf("VaultRelative returned error for POSIX paths: %v", err)
	}
	if rel != "subdir/file.md" {
		t.Fatalf("expected relative path 'subdir/file.md', got %q", rel)
	}

	windowsVault := strings.ReplaceAll(posixVault, string(filepath.Separator), "\\")
	windowsFile := strings.ReplaceAll(posixFile, string(filepath.Separator), "\\")

	rel, err = VaultRelative(windowsVault, windowsFile)
	if err != nil {
		t.Fatalf("VaultRelative returned error for Windows paths: %v", err)
	}
	if rel != "subdir/file.md" {
		t.Fatalf("expected relative path 'subdir/file.md', got %q", rel)
	}
}

func TestVaultRelativeComponentsHandlesRootAndNested(t *testing.T) {
	vault := filepath.Join("vault")
	rootFile := filepath.Join("vault", "root.md")
	nestedFile := filepath.Join("vault", "sub", "dir", "note.md")

	subDir, remainder, err := VaultRelativeComponents(vault, rootFile)
	if err != nil {
		t.Fatalf("VaultRelativeComponents returned error for root file: %v", err)
	}
	if subDir != "" {
		t.Fatalf("expected empty subdir for root file, got %q", subDir)
	}
	if remainder != "root.md" {
		t.Fatalf("expected remainder 'root.md', got %q", remainder)
	}

	subDir, remainder, err = VaultRelativeComponents(vault, nestedFile)
	if err != nil {
		t.Fatalf("VaultRelativeComponents returned error for nested file: %v", err)
	}
	if subDir != "sub" {
		t.Fatalf("expected first subdirectory 'sub', got %q", subDir)
	}
	if remainder != "dir/note.md" {
		t.Fatalf("expected remainder 'dir/note.md', got %q", remainder)
	}
}

func TestVaultRelativeComponentsWithWindowsPaths(t *testing.T) {
	vault := filepath.Join("vault")
	nested := filepath.Join("vault", "folder", "child.md")

	windowsVault := strings.ReplaceAll(vault, string(filepath.Separator), "\\")
	windowsNested := strings.ReplaceAll(nested, string(filepath.Separator), "\\")

	subDir, remainder, err := VaultRelativeComponents(windowsVault, windowsNested)
	if err != nil {
		t.Fatalf("VaultRelativeComponents returned error for Windows paths: %v", err)
	}
	if subDir != "folder" {
		t.Fatalf("expected first subdirectory 'folder', got %q", subDir)
	}
	if remainder != "child.md" {
		t.Fatalf("expected remainder 'child.md', got %q", remainder)
	}
}
