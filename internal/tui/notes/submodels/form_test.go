package submodels

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

func TestHandleSubmitUsesDefaultTemplateWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()

	inputs := make([]textinput.Model, 5)
	inputs[title] = textinput.New()
	inputs[title].SetValue("Test Note")
	inputs[tags] = textinput.New()
	inputs[links] = textinput.New()
	inputs[template] = textinput.New()
	inputs[subdirectory] = textinput.New()
	inputs[subdirectory].SetValue("notes")

	model := FormModel{
		state: &state.State{
			Vault:     tempDir,
			Templater: &templater.Templater{},
		},
		Inputs:           inputs,
		availableSubdirs: []string{"notes"},
	}

	var capturedTemplate string
	originalLauncher := noteLauncher
	noteLauncher = func(_ *note.ZettelkastenNote, _ *templater.Templater, tmpl, _ string) {
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

	if capturedTemplate != "zet" {
		t.Fatalf("expected default template 'zet', got %q", capturedTemplate)
	}

	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}
}
