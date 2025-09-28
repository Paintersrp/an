package parser

import (
	"testing"
	"time"
)

func TestExtractTaskMetadataParsesTokens(t *testing.T) {
	content := "review pull request @due(2024-12-01) @scheduled(2024-11-28) @priority(high) @project(automation) @owner(Sam) [[Release Plan]]"

	cleaned, meta := ExtractTaskMetadata(content)
	if cleaned != "review pull request" {
		t.Fatalf("expected cleaned content, got %q", cleaned)
	}
	if meta.DueDate == nil || meta.DueDate.Format("2006-01-02") != "2024-12-01" {
		t.Fatalf("expected due date to be parsed, got %#v", meta.DueDate)
	}
	if meta.ScheduledDate == nil || meta.ScheduledDate.Format("2006-01-02") != "2024-11-28" {
		t.Fatalf("expected scheduled date to be parsed, got %#v", meta.ScheduledDate)
	}
	if meta.Priority != "high" {
		t.Fatalf("expected priority metadata, got %q", meta.Priority)
	}
	if meta.Project != "automation" {
		t.Fatalf("expected project metadata, got %q", meta.Project)
	}
	if meta.Owner != "Sam" {
		t.Fatalf("expected owner metadata, got %q", meta.Owner)
	}
	if len(meta.References) != 1 || meta.References[0] != "Release Plan" {
		t.Fatalf("expected backlink metadata, got %#v", meta.References)
	}
	if len(meta.RawTokens) != 5 {
		t.Fatalf("expected all tokens to be recorded, got %#v", meta.RawTokens)
	}
}

func TestExtractTaskMetadataPreservesWhitespace(t *testing.T) {
	content := `- [ ] Update docs @due(2024-12-01)
    Continue discussion with team
        subpoint details @priority(high)`

	cleaned, meta := ExtractTaskMetadata(content)

	expected := `- [ ] Update docs 
    Continue discussion with team
        subpoint details`
	if cleaned != expected {
		t.Fatalf("expected cleaned content to preserve whitespace, got %q", cleaned)
	}

	if meta.DueDate == nil {
		t.Fatalf("expected due metadata to be parsed")
	}
	if meta.Priority != "high" {
		t.Fatalf("expected priority metadata to be parsed, got %q", meta.Priority)
	}
}

func TestParseDateRecognizesKeywords(t *testing.T) {
	today := time.Now()
	parsed, ok := parseDate("today")
	if !ok {
		t.Fatalf("expected today keyword to parse")
	}
	if parsed.Year() != today.Year() || parsed.Month() != today.Month() || parsed.Day() != today.Day() {
		t.Fatalf("expected today keyword to use current day, got %v", parsed)
	}
}
