package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Paintersrp/an/internal/config"
)

func writeConfigFile(t *testing.T, home string, data map[string]any) {
	t.Helper()

	configPath := config.GetConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	bytes, err := yaml.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal config data: %v", err)
	}

	if err := os.WriteFile(configPath, bytes, 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
}

func TestLoadAcceptsSupportedEditors(t *testing.T) {
	editors := []string{"nvim", "obsidian", "vscode", "vim", "nano"}

	for _, editor := range editors {
		editor := editor
		t.Run(editor, func(t *testing.T) {
			home := t.TempDir()
			cfgData := map[string]any{
				"current_workspace": "main",
				"workspaces": map[string]any{
					"main": map[string]any{
						"vaultdir": filepath.Join(home, "vault"),
						"editor":   editor,
						"fsmode":   "strict",
					},
				},
			}

			writeConfigFile(t, home, cfgData)

			cfg, err := config.Load(home)
			if err != nil {
				t.Fatalf("expected load to succeed for editor %q: %v", editor, err)
			}

			if got := cfg.MustWorkspace().Editor; got != editor {
				t.Fatalf("expected editor %q, got %q", editor, got)
			}
		})
	}
}

func TestLoadRejectsUnsupportedEditor(t *testing.T) {
	home := t.TempDir()
	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"editor":   "unsupported",
				"fsmode":   "strict",
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	_, err := config.Load(home)
	if err == nil {
		t.Fatal("expected load to fail for unsupported editor")
	}

	if want := "invalid editor"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}

func TestLoadMigratesLegacyConfig(t *testing.T) {
	home := t.TempDir()
	legacy := map[string]any{
		"vaultdir": filepath.Join(home, "vault"),
		"editor":   "vim",
		"fsmode":   "confirm",
		"subdirs":  []string{"atoms"},
	}

	writeConfigFile(t, home, legacy)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.CurrentWorkspace != "default" {
		t.Fatalf("expected default workspace, got %q", cfg.CurrentWorkspace)
	}

	ws := cfg.MustWorkspace()
	if ws.VaultDir != filepath.Join(home, "vault") {
		t.Fatalf("expected migrated vaultdir, got %q", ws.VaultDir)
	}
	if ws.Editor != "vim" {
		t.Fatalf("expected migrated editor, got %q", ws.Editor)
	}
	if ws.FileSystemMode != "confirm" {
		t.Fatalf("expected migrated fs mode, got %q", ws.FileSystemMode)
	}
	if !slices.Contains(ws.SubDirs, "atoms") {
		t.Fatalf("expected migrated subdirs to include atoms: %#v", ws.SubDirs)
	}
}

func TestSaveWithNoEditorSkipsValidation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"fsmode":   "strict",
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.MustWorkspace().Editor != "" {
		t.Fatalf("expected empty editor, got %q", cfg.MustWorkspace().Editor)
	}

	if err := cfg.AddSubdir("atoms"); err != nil {
		t.Fatalf("AddSubdir returned error: %v", err)
	}

	reloaded, err := config.Load(home)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}

	if !slices.Contains(reloaded.MustWorkspace().SubDirs, "atoms") {
		t.Fatalf("expected persisted SubDirs to include 'atoms': %#v", reloaded.MustWorkspace().SubDirs)
	}
}

func TestConfigAddSubdirPersistsChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"editor":   "nvim",
				"fsmode":   "strict",
				"subdirs":  []string{"existing"},
			},
		},
	}
	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if err := cfg.AddSubdir("atoms"); err != nil {
		t.Fatalf("AddSubdir returned error: %v", err)
	}

	if !slices.Contains(cfg.MustWorkspace().SubDirs, "atoms") {
		t.Fatalf("expected in-memory SubDirs to include 'atoms': %#v", cfg.MustWorkspace().SubDirs)
	}

	data, err := os.ReadFile(cfg.GetConfigPath())
	if err != nil {
		t.Fatalf("reading persisted config: %v", err)
	}

	var persisted map[string]any
	if err := yaml.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal persisted config: %v", err)
	}

	workspaces := persisted["workspaces"].(map[string]any)
	main := workspaces["main"].(map[string]any)
	subdirs := main["subdirs"].([]any)

	found := false
	for _, v := range subdirs {
		if v.(string) == "atoms" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected persisted SubDirs to include 'atoms': %#v", subdirs)
	}

	if err := cfg.AddSubdir("atoms"); err == nil {
		t.Fatal("expected error when adding duplicate subdir, got nil")
	}
}

func TestConfigAddAndRemoveView(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"editor":   "vim",
				"fsmode":   "strict",
			},
		},
	}
	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if err := cfg.AddView("custom", config.ViewDefinition{Include: []string{"notes"}}); err != nil {
		t.Fatalf("AddView returned error: %v", err)
	}

	if _, ok := cfg.MustWorkspace().Views["custom"]; !ok {
		t.Fatalf("expected in-memory Views to include 'custom': %#v", cfg.MustWorkspace().Views)
	}

	if !slices.Contains(cfg.MustWorkspace().ViewOrder, "custom") {
		t.Fatalf("expected ViewOrder to include 'custom': %#v", cfg.MustWorkspace().ViewOrder)
	}

	data, err := os.ReadFile(cfg.GetConfigPath())
	if err != nil {
		t.Fatalf("reading persisted config: %v", err)
	}

	var persisted map[string]any
	if err := yaml.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal persisted config: %v", err)
	}

	workspaces := persisted["workspaces"].(map[string]any)
	main := workspaces["main"].(map[string]any)
	views := main["views"].(map[string]any)
	if _, ok := views["custom"]; !ok {
		t.Fatalf("expected persisted Views to include 'custom': %#v", views)
	}

	if err := cfg.RemoveView("custom"); err != nil {
		t.Fatalf("RemoveView returned error: %v", err)
	}

	if _, ok := cfg.MustWorkspace().Views["custom"]; ok {
		t.Fatalf("expected view 'custom' to be removed: %#v", cfg.MustWorkspace().Views)
	}

	if slices.Contains(cfg.MustWorkspace().ViewOrder, "custom") {
		t.Fatalf("expected ViewOrder to exclude 'custom': %#v", cfg.MustWorkspace().ViewOrder)
	}
}

func TestWorkspaceSwitchPersists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgData := map[string]any{
		"current_workspace": "primary",
		"workspaces": map[string]any{
			"primary": map[string]any{
				"vaultdir": filepath.Join(home, "vault1"),
				"editor":   "nvim",
				"fsmode":   "strict",
			},
			"secondary": map[string]any{
				"vaultdir": filepath.Join(home, "vault2"),
				"editor":   "vim",
				"fsmode":   "confirm",
			},
		},
	}
	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if err := cfg.SwitchWorkspace("secondary"); err != nil {
		t.Fatalf("SwitchWorkspace returned error: %v", err)
	}

	if cfg.CurrentWorkspace != "secondary" {
		t.Fatalf("expected current workspace to be secondary, got %q", cfg.CurrentWorkspace)
	}

	ws := cfg.MustWorkspace()
	if ws.VaultDir != filepath.Join(home, "vault2") {
		t.Fatalf("expected active workspace to switch vaultdir, got %q", ws.VaultDir)
	}

	reloaded, err := config.Load(home)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}

	if reloaded.CurrentWorkspace != "secondary" {
		t.Fatalf("expected persisted current workspace to be secondary, got %q", reloaded.CurrentWorkspace)
	}
}
