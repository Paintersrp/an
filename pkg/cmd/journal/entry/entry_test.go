package entry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/services/journal"
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

	ws := &config.Workspace{VaultDir: vaultDir, Editor: "nvim"}
	cfg := &config.Config{
		Workspaces:       map[string]*config.Workspace{"default": ws},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	tmpl, err := templater.NewTemplater(ws)
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	h := handler.NewFileHandler(vaultDir)

	st := &state.State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: "default",
		Templater:     tmpl,
		Handler:       h,
		Vault:         vaultDir,
	}

	return st, vaultDir
}

func TestRunWithTagsOnly(t *testing.T) {
	st, vaultDir := setupEntryTest(t)

	cmd := NewCmdEntry(st, "day")
	args := []string{"test-tag"}

	svc := journal.NewService(st.Templater, st.Handler)
	if err := run(cmd, args, svc, 0, "day"); err != nil {
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

	svc := journal.NewService(st.Templater, st.Handler)
	if err := run(cmd, args, svc, 0, "day"); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	date := utils.GenerateDate(0, "day")
	notePath := filepath.Join(vaultDir, "atoms", fmt.Sprintf("day-%s.md", date))

	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}

	raw := string(content)
	if !strings.Contains(raw, "Inline content! #1") {
		t.Fatalf("expected note to contain inline content, content: %s", string(content))
	}

	frontMatter := strings.SplitN(raw, "---", 2)[0]
	if strings.Contains(frontMatter, "Inline content! #1") {
		t.Fatalf("inline content should not appear in front matter, content: %s", frontMatter)
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

	svc := journal.NewService(st.Templater, st.Handler)
	if err := run(cmd, args, svc, 0, "day"); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	date := utils.GenerateDate(0, "day")
	notePath := filepath.Join(vaultDir, "atoms", fmt.Sprintf("day-%s.md", date))

	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}

	raw := string(content)
	if !strings.Contains(raw, "Pasted content!") {
		t.Fatalf("expected note to contain pasted content, content: %s", string(content))
	}

	frontMatter := strings.SplitN(raw, "---", 2)[0]
	if strings.Contains(frontMatter, "Pasted content!") {
		t.Fatalf("pasted content should not appear in front matter, content: %s", frontMatter)
	}
}
