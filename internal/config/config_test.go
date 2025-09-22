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

func TestLoadAcceptsSupportedEditors(t *testing.T) {
	editors := []string{"nvim", "obsidian", "vscode", "vim", "nano"}

	for _, editor := range editors {
		editor := editor
		t.Run(editor, func(t *testing.T) {
			home := t.TempDir()
			configPath := config.GetConfigPath(home)

			if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
				t.Fatalf("failed to create config directory: %v", err)
			}

			cfgData := map[string]any{
				"vaultdir":         filepath.Join(home, "vault"),
				"editor":           editor,
				"nvimargs":         "",
				"fsmode":           "strict",
				"pinned_file":      "",
				"pinned_task_file": "",
				"subdirs":          []string{},
			}

			data, err := yaml.Marshal(cfgData)
			if err != nil {
				t.Fatalf("failed to marshal config data: %v", err)
			}

			if err := os.WriteFile(configPath, data, 0o644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := config.Load(home)
			if err != nil {
				t.Fatalf("expected load to succeed for editor %q: %v", editor, err)
			}

			if cfg.Editor != editor {
				t.Fatalf("expected editor %q, got %q", editor, cfg.Editor)
			}
		})
	}
}

func TestLoadRejectsUnsupportedEditor(t *testing.T) {
	home := t.TempDir()
	configPath := config.GetConfigPath(home)

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	cfgData := map[string]any{
		"vaultdir":         filepath.Join(home, "vault"),
		"editor":           "unsupported", // ensure validation fails
		"nvimargs":         "",
		"fsmode":           "strict",
		"pinned_file":      "",
		"pinned_task_file": "",
		"subdirs":          []string{},
	}

	data, err := yaml.Marshal(cfgData)
	if err != nil {
		t.Fatalf("failed to marshal config data: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = config.Load(home)
	if err == nil {
		t.Fatal("expected load to fail for unsupported editor")
	}

	if !strings.Contains(err.Error(), "invalid editor") {
		t.Fatalf("expected invalid editor error, got %v", err)
	}
}

func TestConfigAddSubdirPersistsChanges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := &config.Config{
		Editor:         "nvim",
		FileSystemMode: "strict",
		SubDirs:        []string{"existing"},
	}

	if err := cfg.AddSubdir("atoms"); err != nil {
		t.Fatalf("AddSubdir returned error: %v", err)
	}

	if !slices.Contains(cfg.SubDirs, "atoms") {
		t.Fatalf("expected in-memory SubDirs to include 'atoms': %#v", cfg.SubDirs)
	}

	data, err := os.ReadFile(cfg.GetConfigPath())
	if err != nil {
		t.Fatalf("reading persisted config: %v", err)
	}

	var persisted config.Config
	if err := yaml.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal persisted config: %v", err)
	}

	if !slices.Contains(persisted.SubDirs, "atoms") {
		t.Fatalf("expected persisted SubDirs to include 'atoms': %#v", persisted.SubDirs)
	}

	if err := cfg.AddSubdir("atoms"); err == nil {
		t.Fatal("expected error when adding duplicate subdir, got nil")
	}
}
