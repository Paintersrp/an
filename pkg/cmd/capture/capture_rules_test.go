package capture

import (
	"reflect"
	"testing"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
)

func TestResolveCaptureMetadataOverlappingRules(t *testing.T) {
	original := readClipboard
	t.Cleanup(func() {
		readClipboard = original
	})

	readClipboard = func() (string, error) {
		return "clipboard payload", nil
	}

	s := &state.State{
		Workspace: &config.Workspace{
			Capture: config.CaptureConfig{
				Rules: []config.CaptureRule{
					{
						Action: config.CaptureAction{
							Tags:        []string{"global", "link"},
							FrontMatter: map[string]any{"status": "general"},
							Fields:      map[string]any{"status": "general"},
						},
					},
					{
						Match: config.CaptureMatcher{Template: "daily"},
						Action: config.CaptureAction{
							Tags:        []string{"daily", "link"},
							FrontMatter: map[string]any{"status": "daily", "review": true},
							Fields:      map[string]any{"status": "daily"},
						},
					},
					{
						Match: config.CaptureMatcher{UpstreamPrefix: "obsidian://"},
						Action: config.CaptureAction{
							Tags:        []string{"sync"},
							FrontMatter: map[string]any{"status": "synced", "priority": 1},
							Fields:      map[string]any{"status": "synced"},
						},
					},
					{
						Action: config.CaptureAction{
							Clipboard:   true,
							Tags:        []string{"clip", "sync"},
							FrontMatter: map[string]any{"clipboard": "attached"},
							Fields:      map[string]any{"origin": "clipboard"},
						},
					},
				},
			},
		},
	}

	tags, metadata, fields, err := resolveCaptureMetadata(s, "daily", "obsidian://note")
	if err != nil {
		t.Fatalf("resolveCaptureMetadata returned error: %v", err)
	}

	wantTags := []string{"global", "link", "daily", "sync", "clip"}
	if !reflect.DeepEqual(tags, wantTags) {
		t.Fatalf("expected tags %v, got %v", wantTags, tags)
	}

	wantMetadata := map[string]any{
		"status":    "synced",
		"review":    true,
		"priority":  1,
		"clipboard": "attached",
	}
	if !reflect.DeepEqual(metadata, wantMetadata) {
		t.Fatalf("expected metadata %#v, got %#v", wantMetadata, metadata)
	}

	wantFields := map[string]any{
		"origin": "clipboard",
		"status": "synced",
	}
	if !reflect.DeepEqual(fields, wantFields) {
		t.Fatalf("expected fields %#v, got %#v", wantFields, fields)
	}
}

func TestResolveCaptureMetadataNoRules(t *testing.T) {
	s := &state.State{Workspace: &config.Workspace{}}

	tags, metadata, fields, err := resolveCaptureMetadata(s, "", "")
	if err != nil {
		t.Fatalf("resolveCaptureMetadata returned error: %v", err)
	}
	if tags != nil {
		t.Fatalf("expected nil tags, got %v", tags)
	}
	if metadata != nil {
		t.Fatalf("expected nil metadata, got %v", metadata)
	}
	if fields != nil {
		t.Fatalf("expected nil fields, got %v", fields)
	}
}

func TestMergeTagSetsDeduplicatesAutomation(t *testing.T) {
	manual := []string{"manual", "foo"}
	automated := []string{"foo", "clip", "manual", "clip"}

	got := mergeTagSets(manual, automated)
	want := []string{"manual", "foo", "clip"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeTagSets dedupe failed, got %v want %v", got, want)
	}
}
