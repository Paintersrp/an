package templater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTemplaterRegistersUserTemplate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	templatesDir := filepath.Join(os.Getenv("HOME"), ".an", "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("failed to create user template directory: %v", err)
	}

	customTemplatePath := filepath.Join(templatesDir, "custom.tmpl")
	if err := os.WriteFile(customTemplatePath, []byte("Title: {{.Title}}"), 0o644); err != nil {
		t.Fatalf("failed to write user template: %v", err)
	}

	prevValue, hadPrev := AvailableTemplates["custom"]
	defer func() {
		if hadPrev {
			AvailableTemplates["custom"] = prevValue
		} else {
			delete(AvailableTemplates, "custom")
		}
	}()

	tmpl, err := NewTemplater()
	if err != nil {
		t.Fatalf("NewTemplater returned error: %v", err)
	}

	tpl, ok := tmpl.templates["custom"]
	if !ok {
		t.Fatalf("expected custom template to be registered: %#v", tmpl.templates)
	}

	if tpl.FilePath != customTemplatePath {
		t.Fatalf("expected template path %q, got %q", customTemplatePath, tpl.FilePath)
	}
}

func TestTemplateMapLoadTemplates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "user.tmpl"), []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	m := make(TemplateMap)
	if err := m.loadTemplates(dir); err != nil {
		t.Fatalf("loadTemplates returned error: %v", err)
	}

	tpl, ok := m["user"]
	if !ok {
		t.Fatal("expected template named 'user' to be loaded")
	}

	if tpl.FilePath == "" {
		t.Fatal("expected template FilePath to be recorded")
	}
}

func TestGenerateTagsAndDateDefaultTemplate(t *testing.T) {
	t.Setenv("TZ", "UTC")

	templater := &Templater{}
	date, tags := templater.GenerateTagsAndDate("roadmap")

	if len(date) == 0 {
		t.Fatal("expected generated date to be non-empty")
	}

	if len(tags) != 0 {
		t.Fatalf("expected non-daily template to have zero tags, got %#v", tags)
	}
}

func TestGenerateTagsAndDateDayTemplate(t *testing.T) {
	t.Setenv("TZ", "UTC")

	templater := &Templater{}

	before := time.Now().UTC()
	date, tags := templater.GenerateTagsAndDate("day")
	after := time.Now().UTC()

	if len(date) == 0 {
		t.Fatal("expected generated date to be non-empty")
	}

	if len(tags) != 3 {
		t.Fatalf("expected day template to have three tags, got %#v", tags)
	}

	if tags[0] != "daily" {
		t.Fatalf("expected first tag to be 'daily', got %q", tags[0])
	}

	beforeDay := strings.ToLower(before.Weekday().String())
	afterDay := strings.ToLower(after.Weekday().String())
	if tags[1] != beforeDay && tags[1] != afterDay {
		t.Fatalf("expected second tag to match weekday, got %q (expected %q or %q)", tags[1], beforeDay, afterDay)
	}

	beforeHour := fmt.Sprintf("%02dh", before.Hour())
	afterHour := fmt.Sprintf("%02dh", after.Hour())
	if tags[2] != beforeHour && tags[2] != afterHour {
		t.Fatalf("expected third tag to match hour, got %q (expected %q or %q)", tags[2], beforeHour, afterHour)
	}
}

func TestExecuteMissingTemplate(t *testing.T) {
	templater := &Templater{templates: make(TemplateMap)}

	if _, err := templater.Execute("missing", TemplateData{}); err == nil {
		t.Fatal("expected error when executing missing template, got nil")
	}
}

func TestTemplaterExecuteRendersTemplate(t *testing.T) {
	templater := &Templater{templates: TemplateMap{
		"custom": {
			Content: "Title: {{.Title}}",
		},
	}}

	rendered, err := templater.Execute("custom", TemplateData{Title: "Rendered"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !strings.Contains(rendered, "Rendered") {
		t.Fatalf("expected rendered template to include title, got %q", rendered)
	}
}

func TestTemplaterExecuteUsesUserTemplateContent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	templatesDir := filepath.Join(os.Getenv("HOME"), ".an", "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("failed to create user template directory: %v", err)
	}

	const templateBody = "User template body"

	customTemplatePath := filepath.Join(templatesDir, "custom.tmpl")
	if err := os.WriteFile(customTemplatePath, []byte(templateBody), 0o644); err != nil {
		t.Fatalf("failed to write user template: %v", err)
	}

	prevValue, hadPrev := AvailableTemplates["custom"]
	defer func() {
		if hadPrev {
			AvailableTemplates["custom"] = prevValue
		} else {
			delete(AvailableTemplates, "custom")
		}
	}()

	templater, err := NewTemplater()
	if err != nil {
		t.Fatalf("NewTemplater returned error: %v", err)
	}

	rendered, err := templater.Execute("custom", TemplateData{})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if rendered != templateBody {
		t.Fatalf("expected rendered template to match template body %q, got %q", templateBody, rendered)
	}
}
