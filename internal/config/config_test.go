package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Paintersrp/an/internal/config"
	"github.com/spf13/viper"
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

func TestLoadSetsReviewDefaults(t *testing.T) {
	home := t.TempDir()
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

	ws := cfg.MustWorkspace()
	if !ws.Review.Enable {
		t.Fatalf("expected review to be enabled by default")
	}
	if got := ws.Review.Directory; got != "reviews" {
		t.Fatalf("expected review directory 'reviews', got %q", got)
	}
}

func TestLoadReviewDirectoryPrefersLegacyPin(t *testing.T) {
	home := t.TempDir()
	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir":   filepath.Join(home, "vault"),
				"fsmode":     "strict",
				"named_pins": map[string]any{"review": "rituals"},
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	ws := cfg.MustWorkspace()
	if got := ws.Review.Directory; got != "rituals" {
		t.Fatalf("expected review directory 'rituals', got %q", got)
	}
}

func TestLoadReviewOverrides(t *testing.T) {
	home := t.TempDir()
	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"fsmode":   "strict",
				"review": map[string]any{
					"enable":    false,
					"directory": "custom/logs",
				},
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	ws := cfg.MustWorkspace()
	if ws.Review.Enable {
		t.Fatalf("expected review to be disabled")
	}
	if got := ws.Review.Directory; got != "custom/logs" {
		t.Fatalf("expected review directory 'custom/logs', got %q", got)
	}
}

func TestLoadCaptureRules(t *testing.T) {
	home := t.TempDir()
	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"fsmode":   "strict",
				"capture": map[string]any{
					"rules": []any{
						map[string]any{
							"match": map[string]any{
								"template":        "daily",
								"upstream_prefix": "obsidian://",
							},
							"action": map[string]any{
								"clipboard":    true,
								"tags":         []any{"foo", "bar"},
								"front_matter": map[string]any{"status": "wip", "priority": 2},
								"fields":       map[string]any{"source": "rule"},
							},
						},
					},
				},
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	ws := cfg.MustWorkspace()
	if got := len(ws.Capture.Rules); got != 1 {
		t.Fatalf("expected 1 capture rule, got %d", got)
	}

	rule := ws.Capture.Rules[0]
	if rule.Match.Template != "daily" {
		t.Fatalf("expected template 'daily', got %q", rule.Match.Template)
	}
	if rule.Match.UpstreamPrefix != "obsidian://" {
		t.Fatalf("expected upstream prefix 'obsidian://', got %q", rule.Match.UpstreamPrefix)
	}
	if !rule.Action.Clipboard {
		t.Fatalf("expected clipboard action to be true")
	}
	if !slices.Equal(rule.Action.Tags, []string{"foo", "bar"}) {
		t.Fatalf("expected tags [foo bar], got %v", rule.Action.Tags)
	}

	wantFrontMatter := map[string]any{"status": "wip", "priority": 2}
	if !reflect.DeepEqual(rule.Action.FrontMatter, wantFrontMatter) {
		t.Fatalf("expected front matter %#v, got %#v", wantFrontMatter, rule.Action.FrontMatter)
	}

	wantFields := map[string]any{"source": "rule"}
	if !reflect.DeepEqual(rule.Action.Fields, wantFields) {
		t.Fatalf("expected fields %#v, got %#v", wantFields, rule.Action.Fields)
	}
}

func TestLoadCaptureDefaults(t *testing.T) {
	home := t.TempDir()
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

	ws := cfg.MustWorkspace()
	if ws.Capture.Rules == nil {
		t.Fatalf("expected capture rules slice to be initialized")
	}
	if len(ws.Capture.Rules) != 0 {
		t.Fatalf("expected no capture rules by default, got %d", len(ws.Capture.Rules))
	}
}

func TestLoadCaptureRulesMultipleDefinitions(t *testing.T) {
	home := t.TempDir()
	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"fsmode":   "strict",
				"capture": map[string]any{
					"rules": []any{
						map[string]any{
							"match": map[string]any{
								"template": "daily",
							},
							"action": map[string]any{
								"tags": []any{"daily"},
							},
						},
						map[string]any{
							"match": map[string]any{
								"upstream_prefix": "obsidian://",
							},
							"action": map[string]any{
								"clipboard":    true,
								"front_matter": map[string]any{"status": "synced"},
							},
						},
					},
				},
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	rules := cfg.MustWorkspace().Capture.Rules
	if got := len(rules); got != 2 {
		t.Fatalf("expected 2 capture rules, got %d", got)
	}

	if got := rules[0].Match.Template; got != "daily" {
		t.Fatalf("expected first rule template 'daily', got %q", got)
	}
	if len(rules[0].Action.Tags) != 1 || rules[0].Action.Tags[0] != "daily" {
		t.Fatalf("expected first rule tags [daily], got %v", rules[0].Action.Tags)
	}

	if got := rules[1].Match.UpstreamPrefix; got != "obsidian://" {
		t.Fatalf("expected second rule upstream prefix 'obsidian://', got %q", got)
	}
	if rules[1].Action.Clipboard != true {
		t.Fatalf("expected clipboard action enabled for second rule")
	}
	if len(rules[1].Action.Tags) != 0 {
		t.Fatalf("expected second rule tags to default empty, got %v", rules[1].Action.Tags)
	}
	wantFrontMatter := map[string]any{"status": "synced"}
	if !reflect.DeepEqual(rules[1].Action.FrontMatter, wantFrontMatter) {
		t.Fatalf("expected front matter %#v, got %#v", wantFrontMatter, rules[1].Action.FrontMatter)
	}
}

func TestLoadCaptureRulesSectionInitialisesSlice(t *testing.T) {
	home := t.TempDir()
	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"fsmode":   "strict",
				"capture":  map[string]any{},
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	rules := cfg.MustWorkspace().Capture.Rules
	if rules == nil {
		t.Fatalf("expected capture rules slice to be initialised")
	}
	if len(rules) != 0 {
		t.Fatalf("expected no capture rules, got %d", len(rules))
	}
}

func TestCaptureRulesRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir": filepath.Join(home, "vault"),
				"fsmode":   "strict",
				"capture": map[string]any{
					"rules": []any{
						map[string]any{
							"match": map[string]any{
								"template":        "daily",
								"upstream_prefix": "obsidian://",
							},
							"action": map[string]any{
								"clipboard":    true,
								"tags":         []any{"foo", "bar"},
								"front_matter": map[string]any{"status": "wip", "priority": 2},
								"fields":       map[string]any{"source": "rule"},
							},
						},
					},
				},
			},
		},
	}

	writeConfigFile(t, home, cfgData)

	cfg, err := config.Load(home)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	reloaded, err := config.Load(home)
	if err != nil {
		t.Fatalf("reload after save failed: %v", err)
	}

	want := []config.CaptureRule{
		{
			Match: config.CaptureMatcher{
				Template:       "daily",
				UpstreamPrefix: "obsidian://",
			},
			Action: config.CaptureAction{
				Clipboard: true,
				Tags:      []string{"foo", "bar"},
				FrontMatter: map[string]any{
					"status":   "wip",
					"priority": 2,
				},
				Fields: map[string]any{
					"source": "rule",
				},
			},
		},
	}

	if !reflect.DeepEqual(reloaded.MustWorkspace().Capture.Rules, want) {
		t.Fatalf("capture rules did not round-trip: got %#v, want %#v", reloaded.MustWorkspace().Capture.Rules, want)
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

func TestActiveWorkspaceSetsViperValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgData := map[string]any{
		"current_workspace": "main",
		"workspaces": map[string]any{
			"main": map[string]any{
				"vaultdir":         filepath.Join(home, "vault"),
				"editor":           "nvim",
				"nvimargs":         "--clean",
				"fsmode":           "strict",
				"pinned_file":      "pins.md",
				"pinned_task_file": "tasks.md",
				"subdirs":          []string{"atoms", "projects"},
			},
		},
	}

	writeConfigFile(t, home, cfgData)
	viper.Reset()

	if _, err := config.Load(home); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	wantVault := filepath.Join(home, "vault")
	if got := viper.GetString("vaultdir"); got != wantVault {
		t.Fatalf("expected vaultdir %q, got %q", wantVault, got)
	}
	if got := viper.GetString("vaultDir"); got != wantVault {
		t.Fatalf("expected vaultDir %q, got %q", wantVault, got)
	}
	if got := viper.GetString("editor"); got != "nvim" {
		t.Fatalf("expected editor 'nvim', got %q", got)
	}
	if got := viper.GetString("nvimargs"); got != "--clean" {
		t.Fatalf("expected nvimargs '--clean', got %q", got)
	}
	if got := viper.GetString("fsmode"); got != "strict" {
		t.Fatalf("expected fsmode 'strict', got %q", got)
	}
	wantSubdirs := []string{"atoms", "projects"}
	if got := viper.GetStringSlice("subdirs"); !slices.Equal(got, wantSubdirs) {
		t.Fatalf("expected subdirs %v, got %v", wantSubdirs, got)
	}
	if got := viper.GetString("pinned_file"); got != "pins.md" {
		t.Fatalf("expected pinned_file 'pins.md', got %q", got)
	}
	if got := viper.GetString("pinned_task_file"); got != "tasks.md" {
		t.Fatalf("expected pinned_task_file 'tasks.md', got %q", got)
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
