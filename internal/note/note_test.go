package note

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/templater"
)

func TestCreateCleansUpOnTemplateError(t *testing.T) {
	t.Parallel()

	vaultDir := t.TempDir()
	subDir := filepath.Join("foo", "bar")
	note := NewZettelkastenNote(vaultDir, subDir, "test-note", nil, nil, "")

	tmpl, err := templater.NewTemplater(nil)
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	created, err := note.Create("nonexistent-template", tmpl, "", nil)
	if err == nil {
		t.Fatalf("expected template execution error, got nil")
	}

	if created {
		t.Fatalf("expected note creation to fail")
	}

	if _, err := os.Stat(note.GetFilepath()); !os.IsNotExist(err) {
		t.Fatalf("expected note file to be removed, got err %v", err)
	}

	deepestDir := filepath.Join(vaultDir, subDir)
	if _, err := os.Stat(deepestDir); !os.IsNotExist(err) {
		t.Fatalf("expected deepest directory to be removed, got err %v", err)
	}

	parentDir := filepath.Join(vaultDir, "foo")
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Fatalf("expected parent directory to be removed, got err %v", err)
	}
}

func TestEditorLaunchWithTemplateWrapsDefaultCommand(t *testing.T) {
	t.Parallel()
	viper.Reset()
	t.Cleanup(viper.Reset)

	vaultDir := t.TempDir()
	notePath := filepath.Join(vaultDir, "note.md")

	viper.Set("vaultdir", vaultDir)
	viper.Set("editor", "nvim")
	viper.Set("nvimargs", "--headless")
	viper.Set("editor_template", config.CommandTemplate{
		Exec: "kitty",
		Args: []string{"@", "launch", "--type=tab", "{cmd}", "{args}"},
	})

	launch, err := EditorLaunchForPath(notePath, false)
	if err != nil {
		t.Fatalf("unexpected error resolving launch: %v", err)
	}

	expected := []string{"kitty", "@", "launch", "--type=tab", "nvim", "--headless", notePath}
	if !reflect.DeepEqual(launch.Cmd.Args, expected) {
		t.Fatalf("unexpected command args: %#v", launch.Cmd.Args)
	}

	if !launch.Wait {
		t.Fatalf("expected template to inherit wait=true from base editor")
	}
}

func TestEditorLaunchWithCustomTemplate(t *testing.T) {
	t.Parallel()
	viper.Reset()
	t.Cleanup(viper.Reset)

	vaultDir := t.TempDir()
	notePath := filepath.Join(vaultDir, "note.md")

	viper.Set("vaultdir", vaultDir)
	viper.Set("editor", "custom")
	wait := false
	viper.Set("editor_template", config.CommandTemplate{
		Exec: "zed",
		Args: []string{"--reuse-window", "{file}"},
		Wait: &wait,
	})

	launch, err := EditorLaunchForPath(notePath, false)
	if err != nil {
		t.Fatalf("unexpected error resolving launch: %v", err)
	}

	expected := []string{"zed", "--reuse-window", notePath}
	if !reflect.DeepEqual(launch.Cmd.Args, expected) {
		t.Fatalf("unexpected command args: %#v", launch.Cmd.Args)
	}

	if launch.Wait {
		t.Fatalf("expected template wait override to disable waiting")
	}
}

func TestCollectTemplateMetadataPrefillSingle(t *testing.T) {
	templater := newTestTemplater(t)

	metadata, err := CollectTemplateMetadataNonInteractive(templater, "test", map[string]any{
		"status": "done",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	status, ok := metadata["status"].(string)
	if !ok {
		t.Fatalf("expected status to be a string, got %T", metadata["status"])
	}
	if status != "done" {
		t.Fatalf("expected status to be 'done', got %q", status)
	}
}

func TestCollectTemplateMetadataPrefillMulti(t *testing.T) {
	templater := newTestTemplater(t)

	metadata, err := CollectTemplateMetadataNonInteractive(templater, "test", map[string]any{
		"status": "todo",
		"tags":   []interface{}{"one", "two"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tags, ok := metadata["tags"].([]string)
	if !ok {
		t.Fatalf("expected tags to be []string, got %T", metadata["tags"])
	}
	expected := []string{"one", "two"}
	if !reflect.DeepEqual(tags, expected) {
		t.Fatalf("expected tags %v, got %v", expected, tags)
	}
}

func TestCollectTemplateMetadataPrefillRequiredEmpty(t *testing.T) {
	templater := newTestTemplater(t)

	_, err := CollectTemplateMetadataNonInteractive(templater, "test", map[string]any{
		"status": "",
	})
	if err == nil {
		t.Fatalf("expected error for empty required field")
	}
}

func TestCollectTemplateMetadataPrefillUnknownKey(t *testing.T) {
	templater := newTestTemplater(t)

	_, err := CollectTemplateMetadataNonInteractive(templater, "test", map[string]any{
		"status":  "todo",
		"unknown": "value",
	})
	if err == nil {
		t.Fatalf("expected error for unknown prefill key")
	}
}

func TestCollectTemplateMetadataPrefillInvalidType(t *testing.T) {
	templater := newTestTemplater(t)

	_, err := CollectTemplateMetadataNonInteractive(templater, "test", map[string]any{
		"status": "todo",
		"tags":   5,
	})
	if err == nil {
		t.Fatalf("expected error for invalid prefill type")
	}
}

func newTestTemplater(t *testing.T) *templater.Templater {
	t.Helper()

	dir := t.TempDir()
	templatesDir := filepath.Join(dir, ".an", "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	content := `{{/* an:manifest
name: test
fields:
  - key: status
    prompt: Status
    required: true
    options: ["todo", "done"]
  - key: tags
    prompt: Tags
    multi: true
    options: ["one", "two", "three"]
*/}}
Body
`

	if err := os.WriteFile(filepath.Join(templatesDir, "test.tmpl"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	templater, err := templater.NewTemplater(&config.Workspace{VaultDir: dir})
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	return templater
}
