package submodels

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

func TestHandleSubmitCreatesNoteWithBlankSubdirectory(t *testing.T) {
	tempVault := t.TempDir()

	stubDir := t.TempDir()
	editorPath := filepath.Join(stubDir, "nvim")
	if err := os.WriteFile(editorPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to create stub editor: %v", err)
	}

	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", stubDir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", originalPath)
	})

	originalEditor := viper.GetString("editor")
	originalVault := viper.GetString("vaultdir")
	viper.Set("editor", "nvim")
	viper.Set("vaultdir", tempVault)
	t.Cleanup(func() {
		viper.Set("editor", originalEditor)
		viper.Set("vaultdir", originalVault)
	})

	tmpl, err := templater.NewTemplater()
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	testState := &state.State{
		Vault:     tempVault,
		Handler:   handler.NewFileHandler(tempVault),
		Templater: tmpl,
	}

	form := NewFormModel(testState)
	form.Inputs[title].SetValue("blank-subdir-note")
	form.Inputs[template].SetValue("zet")
	form.Inputs[subdirectory].SetValue("")

	form = form.handleSubmit()

	notePath := filepath.Join(tempVault, "blank-subdir-note.md")
	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("expected note file to be created, but got error: %v", err)
	}
}
