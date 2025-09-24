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
