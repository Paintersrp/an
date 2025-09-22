package entry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/utils"
)

func setupEntryTest(t *testing.T) (*state.State, string) {
	t.Helper()

	t.Setenv("HOME", t.TempDir())

	viper.Reset()
	t.Cleanup(viper.Reset)

	vaultDir := t.TempDir()
	viper.Set("vaultdir", vaultDir)
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

	st := &state.State{
		Templater: tmpl,
	}

	return st, vaultDir
}

func TestRunWithTagsOnly(t *testing.T) {
	st, vaultDir := setupEntryTest(t)

	cmd := NewCmdEntry(st, "day")
	args := []string{"test-tag"}

	if err := run(cmd, args, st.Templater, 0, "day"); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	date := utils.GenerateDate(0, "day")
	notePath := filepath.Join(vaultDir, "atoms", fmt.Sprintf("day-%s.md", date))

	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}

	if !strings.Contains(string(content), "- test-tag") {
		t.Fatalf("expected note to contain tag, content: %s", string(content))
	}
}

func TestRunWithTagsAndInlineContent(t *testing.T) {
	st, vaultDir := setupEntryTest(t)

	cmd := NewCmdEntry(st, "day")
	args := []string{"inline-tag", "Inline content! #1"}

	if err := run(cmd, args, st.Templater, 0, "day"); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	date := utils.GenerateDate(0, "day")
	notePath := filepath.Join(vaultDir, "atoms", fmt.Sprintf("day-%s.md", date))

	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}

	if !strings.Contains(string(content), "Inline content! #1") {
		t.Fatalf("expected note to contain inline content, content: %s", string(content))
	}
}

func TestRunWithPasteContent(t *testing.T) {
	st, vaultDir := setupEntryTest(t)

	cmd := NewCmdEntry(st, "day")
	if err := cmd.Flags().Set("paste", "true"); err != nil {
		t.Fatalf("failed to set paste flag: %v", err)
	}

	originalReadClipboard := readClipboard
	readClipboard = func() (string, error) {
		return "Pasted content!", nil
	}
	t.Cleanup(func() {
		readClipboard = originalReadClipboard
	})

	args := []string{"paste-tag"}

	if err := run(cmd, args, st.Templater, 0, "day"); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	date := utils.GenerateDate(0, "day")
	notePath := filepath.Join(vaultDir, "atoms", fmt.Sprintf("day-%s.md", date))

	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}

	if !strings.Contains(string(content), "Pasted content!") {
		t.Fatalf("expected note to contain pasted content, content: %s", string(content))
	}
}
