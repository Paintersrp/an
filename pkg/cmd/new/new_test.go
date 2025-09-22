package new

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/pin"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

func TestRunCreatesSingleSymlink(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	vaultDir := t.TempDir()
	viper.Set("vaultdir", vaultDir)
	viper.Set("subdir", "")
	viper.Set("subdirs", []string{""})
	viper.Set("editor", "nvim")
	viper.Set("nvimargs", "")

	binDir := t.TempDir()
	nvimPath := filepath.Join(binDir, "nvim")
	if err := os.WriteFile(nvimPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to create nvim stub: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tmpl, err := templater.NewTemplater()
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	cfg := &config.Config{
		VaultDir:       vaultDir,
		Editor:         "nvim",
		FileSystemMode: "strict",
		PinManager: pin.NewPinManager(
			pin.PinMap{},
			pin.PinMap{},
			"",
			"",
		),
		NamedPins:     config.PinMap{},
		NamedTaskPins: config.PinMap{},
	}

	st := &state.State{
		Config:    cfg,
		Templater: tmpl,
	}

	workingDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(originalWD)
	})

	cmd := NewCmdNew(st)
	if err := cmd.Flags().Set("symlink", "true"); err != nil {
		t.Fatalf("failed to set symlink flag: %v", err)
	}

	if err := run(cmd, []string{"test-note"}, st); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	notePath := filepath.Join(vaultDir, "test-note.md")
	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("expected note file at %s: %v", notePath, err)
	}

	entries, err := os.ReadDir(workingDir)
	if err != nil {
		t.Fatalf("failed to read working directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry in working directory, got %d", len(entries))
	}

	symlinkPath := filepath.Join(workingDir, "test-note.md")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("failed to stat symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", symlinkPath)
	}

	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("failed to read symlink target: %v", err)
	}
	expectedTarget := filepath.Join(vaultDir, "test-note.md")
	if target != expectedTarget {
		t.Fatalf("expected symlink target %s, got %s", expectedTarget, target)
	}
}
