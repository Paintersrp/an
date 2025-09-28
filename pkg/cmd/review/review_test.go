package review

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

func TestReviewCommand_WritesLog(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	vault := t.TempDir()
	queueDir := filepath.Join(vault, "logs")

	notes := []struct {
		name string
		mod  time.Time
	}{
		{name: "first-note.md", mod: time.Date(2022, 3, 14, 9, 0, 0, 0, time.UTC)},
		{name: "second-note.md", mod: time.Date(2022, 4, 20, 15, 30, 0, 0, time.UTC)},
	}
	for _, note := range notes {
		path := filepath.Join(vault, note.name)
		if err := os.WriteFile(path, []byte("# Note\n"), 0o644); err != nil {
			t.Fatalf("failed to write note %q: %v", note.name, err)
		}
		if err := os.Chtimes(path, note.mod, note.mod); err != nil {
			t.Fatalf("failed to set mod time for %q: %v", note.name, err)
		}
	}

	cfg := &config.Config{
		Workspaces: map[string]*config.Workspace{
			"default": {
				VaultDir:  vault,
				NamedPins: config.PinMap{"review": "logs"},
			},
		},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	tmpl, err := templater.NewTemplater(cfg.MustWorkspace())
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	st := &state.State{Config: cfg, Workspace: cfg.MustWorkspace(), WorkspaceName: cfg.CurrentWorkspace, Templater: tmpl, Vault: vault}

	cmd := NewCmdReview(st)
	cmd.SetArgs([]string{})
	cmd.SetIn(strings.NewReader("inbox cleared\nfocus planned\ntasks reviewed\n"))
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("review command returned error: %v\noutput: %s", err, output.String())
	}

	entries, err := os.ReadDir(queueDir)
	if err != nil {
		t.Fatalf("failed to read log directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log file, found %d", len(entries))
	}

	logPath := filepath.Join(queueDir, entries[0].Name())
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	logStr := string(content)

	if !strings.Contains(logStr, "### Checklist responses") {
		t.Fatalf("log file missing checklist heading: %s", logStr)
	}
	if !strings.Contains(logStr, "- **Clear capture inbox:** inbox cleared") {
		t.Fatalf("log file missing first response: %s", logStr)
	}
	if !strings.Contains(logStr, "- **Plan focus blocks:** focus planned") {
		t.Fatalf("log file missing second response: %s", logStr)
	}
	if !strings.Contains(logStr, "- **Sweep lingering todos:** tasks reviewed") {
		t.Fatalf("log file missing third response: %s", logStr)
	}
	if !strings.Contains(logStr, "### Resurfacing queue") {
		t.Fatalf("log file missing queue heading: %s", logStr)
	}
	if !strings.Contains(logStr, "first-note.md — last touched 2022-03-14") {
		t.Fatalf("log file missing first queue entry: %s", logStr)
	}
	if !strings.Contains(logStr, "second-note.md — last touched 2022-04-20") {
		t.Fatalf("log file missing second queue entry: %s", logStr)
	}
	if !strings.Contains(output.String(), "Review log saved:") {
		t.Fatalf("expected command output to mention saved log, got: %s", output.String())
	}
}

func TestReviewCommand_LogWriteFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	vault := t.TempDir()
	protected := filepath.Join(vault, "protected")
	if err := os.WriteFile(protected, []byte("occupied"), 0o644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	notePath := filepath.Join(vault, "retro.md")
	if err := os.WriteFile(notePath, []byte("# Retro\n"), 0o644); err != nil {
		t.Fatalf("failed to write note: %v", err)
	}
	old := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(notePath, old, old); err != nil {
		t.Fatalf("failed to set note mod time: %v", err)
	}

	cfg := &config.Config{
		Workspaces: map[string]*config.Workspace{
			"default": {
				VaultDir:  vault,
				NamedPins: config.PinMap{"review": "protected"},
			},
		},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	tmpl, err := templater.NewTemplater(cfg.MustWorkspace())
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	st := &state.State{Config: cfg, Workspace: cfg.MustWorkspace(), WorkspaceName: cfg.CurrentWorkspace, Templater: tmpl, Vault: vault}

	cmd := NewCmdReview(st)
	cmd.SetArgs([]string{"--log-path", "protected"})
	cmd.SetIn(strings.NewReader("done\ndone\ndone\n"))
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err = cmd.Execute()
	if err == nil {
		t.Fatalf("expected error when log directory unavailable")
	}
	if !strings.Contains(err.Error(), "prepare review log directory") {
		t.Fatalf("expected error to mention directory preparation, got: %v", err)
	}
}
