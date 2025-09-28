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
						},
					},
					{
						Match: config.CaptureMatcher{Template: "daily"},
						Action: config.CaptureAction{
							Tags:        []string{"daily", "link"},
							FrontMatter: map[string]any{"status": "daily", "review": true},
						},
					},
					{
						Match: config.CaptureMatcher{UpstreamPrefix: "obsidian://"},
						Action: config.CaptureAction{
							Tags:        []string{"sync"},
							FrontMatter: map[string]any{"status": "synced", "priority": 1},
						},
					},
					{
						Action: config.CaptureAction{
							Clipboard:   true,
							Tags:        []string{"clip", "sync"},
							FrontMatter: map[string]any{"clipboard": "attached"},
						},
					},
				},
			},
		},
	}

	tags, metadata, err := resolveCaptureMetadata(s, "daily", "obsidian://note")
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
}

func TestResolveCaptureMetadataNoRules(t *testing.T) {
	s := &state.State{Workspace: &config.Workspace{}}

	tags, metadata, err := resolveCaptureMetadata(s, "", "")
	if err != nil {
		t.Fatalf("resolveCaptureMetadata returned error: %v", err)
	}
	if tags != nil {
		t.Fatalf("expected nil tags, got %v", tags)
	}
	if metadata != nil {
		t.Fatalf("expected nil metadata, got %v", metadata)
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
