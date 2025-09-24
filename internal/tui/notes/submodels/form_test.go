package submodels

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

func newTestTemplater(t *testing.T) *templater.Templater {
	t.Helper()
	tmpl, err := templater.NewTemplater(nil)
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}
	return tmpl
}

func TestHandleSubmitUsesDefaultTemplateWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()

	inputs := make([]textinput.Model, 5)
	inputs[title] = textinput.New()
	inputs[title].SetValue("Test Note")
	inputs[tags] = textinput.New()
	inputs[links] = textinput.New()
	inputs[template] = textinput.New()
	inputs[template].SetValue("   ")
	inputs[subdirectory] = textinput.New()
	inputs[subdirectory].SetValue("notes")

	model := FormModel{
		state: &state.State{
			Vault:     tempDir,
			Templater: newTestTemplater(t),
		},
		Inputs:           inputs,
		availableSubdirs: []string{"notes"},
	}

	var capturedTemplate string
	originalLauncher := noteLauncher
	noteLauncher = func(_ *note.ZettelkastenNote, _ *templater.Templater, tmpl string, _ string, _ map[string]interface{}) {
		capturedTemplate = tmpl
	}
	defer func() {
		noteLauncher = originalLauncher
	}()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	model.handleSubmit()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	_ = r.Close()
	output := buf.String()

	if capturedTemplate != defaultTemplate {
		t.Fatalf("expected default template %q, got %q", defaultTemplate, capturedTemplate)
	}

	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}
}

func TestHandleSubmitAllowsEmptySubdirectory(t *testing.T) {
	tempDir := t.TempDir()

	inputs := make([]textinput.Model, 5)
	inputs[title] = textinput.New()
	inputs[title].SetValue("Test Note")
	inputs[tags] = textinput.New()
	inputs[links] = textinput.New()
	inputs[template] = textinput.New()
	inputs[subdirectory] = textinput.New()
	inputs[subdirectory].SetValue("")

	model := FormModel{
		state: &state.State{
			Vault:     tempDir,
			Templater: newTestTemplater(t),
		},
		Inputs:           inputs,
		availableSubdirs: []string{"notes"},
	}

	var capturedNote *note.ZettelkastenNote
	originalLauncher := noteLauncher
	noteLauncher = func(n *note.ZettelkastenNote, _ *templater.Templater, _ string, _ string, _ map[string]interface{}) {
		capturedNote = n
	}
	defer func() {
		noteLauncher = originalLauncher
	}()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	model.handleSubmit()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	_ = r.Close()
	output := buf.String()

	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}

	if capturedNote == nil {
		t.Fatalf("expected note to be created, but noteLauncher was not called")
	}

	if capturedNote.SubDir != "" {
		t.Fatalf("expected empty subdirectory, got %q", capturedNote.SubDir)
	}
}

func TestHandleSubmitAllowsLongTitle(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tempDir, "notes"), 0o755); err != nil {
		t.Fatalf("failed to create notes directory: %v", err)
	}

	model := NewFormModel(&state.State{
		Vault:     tempDir,
		Templater: newTestTemplater(t),
	})

	longTitle := strings.Repeat("Long title ", 3) + "with more"
	if len(longTitle) <= 20 {
		t.Fatalf("expected longTitle to be longer than 20 characters, got %d", len(longTitle))
	}

	model.Inputs[title].SetValue(longTitle)
	model.Inputs[tags].SetValue("")
	model.Inputs[links].SetValue("")
	model.Inputs[template].SetValue(defaultTemplate)
	model.Inputs[subdirectory].SetValue("notes")

	var capturedNote *note.ZettelkastenNote
	var capturedTemplate string
	originalLauncher := noteLauncher
	noteLauncher = func(n *note.ZettelkastenNote, _ *templater.Templater, tmpl string, _ string, _ map[string]interface{}) {
		capturedNote = n
		capturedTemplate = tmpl
	}
	defer func() {
		noteLauncher = originalLauncher
	}()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	model.handleSubmit()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	_ = r.Close()
	output := buf.String()

	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}

	if capturedNote == nil {
		t.Fatalf("expected note to be created, but noteLauncher was not called")
	}

	if capturedNote.Filename != longTitle {
		t.Fatalf("expected title %q, got %q", longTitle, capturedNote.Filename)
	}

	if capturedTemplate != defaultTemplate {
		t.Fatalf("expected template %q, got %q", defaultTemplate, capturedTemplate)
	}
}

func TestHandleSubmitAllowsNestedSubdirectory(t *testing.T) {
	tempDir := t.TempDir()

	nested := filepath.Join("notes", "nested")
	if err := os.MkdirAll(filepath.Join(tempDir, nested), 0o755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	inputs := make([]textinput.Model, 5)
	inputs[title] = textinput.New()
	inputs[title].SetValue("Test Note")
	inputs[tags] = textinput.New()
	inputs[links] = textinput.New()
	inputs[template] = textinput.New()
	inputs[template].SetValue(defaultTemplate)
	inputs[subdirectory] = textinput.New()
	inputs[subdirectory].SetValue(nested)

	model := FormModel{
		state: &state.State{
			Vault:     tempDir,
			Templater: newTestTemplater(t),
		},
		Inputs:           inputs,
		availableSubdirs: []string{"notes"},
	}

	var capturedNote *note.ZettelkastenNote
	originalLauncher := noteLauncher
	noteLauncher = func(n *note.ZettelkastenNote, _ *templater.Templater, _ string, _ string, _ map[string]interface{}) {
		capturedNote = n
	}
	defer func() {
		noteLauncher = originalLauncher
	}()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	model.handleSubmit()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	_ = r.Close()
	output := buf.String()

	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}

	if capturedNote == nil {
		t.Fatalf("expected note to be created, but noteLauncher was not called")
	}

	if capturedNote.SubDir != nested {
		t.Fatalf("expected subdirectory %q, got %q", nested, capturedNote.SubDir)
	}
}

func TestNewFormModelIncludesNestedSubdirectories(t *testing.T) {
	tempDir := t.TempDir()

	expected := []string{
		filepath.Join("notes"),
		filepath.Join("notes", "nested"),
	}

	for _, dir := range expected {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Hidden directories should not be suggested.
	hidden := filepath.Join(tempDir, ".hidden")
	if err := os.MkdirAll(hidden, 0o755); err != nil {
		t.Fatalf("failed to create hidden directory: %v", err)
	}

	model := NewFormModel(&state.State{Vault: tempDir})

	for _, dir := range expected {
		found := false
		for _, available := range model.availableSubdirs {
			if available == dir {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected subdirectory %q to be listed, but it was not", dir)
		}
	}

	for _, dir := range model.availableSubdirs {
		if strings.HasPrefix(filepath.Base(dir), ".") {
			t.Fatalf("expected hidden directories to be excluded, but found %q", dir)
		}
	}
}
