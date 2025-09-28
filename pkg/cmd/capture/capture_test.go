package capture

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
)

func TestResolveCaptureMetadataNilWorkspace(t *testing.T) {
	tags, metadata, err := resolveCaptureMetadata(nil, "daily", "")
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

func TestResolveCaptureMetadataMergesRules(t *testing.T) {
	s := &state.State{
		Workspace: &config.Workspace{
			Capture: config.CaptureConfig{
				Rules: []config.CaptureRule{
					{
						Match: config.CaptureMatcher{Template: "daily"},
						Action: config.CaptureAction{
							Tags:        []string{"foo"},
							FrontMatter: map[string]any{"status": "wip"},
						},
					},
					{
						Match: config.CaptureMatcher{UpstreamPrefix: "obsidian://"},
						Action: config.CaptureAction{
							Tags:        []string{"foo", "bar"},
							FrontMatter: map[string]any{"priority": 2},
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

	wantTags := []string{"foo", "bar"}
	if !reflect.DeepEqual(tags, wantTags) {
		t.Fatalf("expected tags %v, got %v", wantTags, tags)
	}

	wantMetadata := map[string]any{
		"status":   "wip",
		"priority": 2,
	}
	if !reflect.DeepEqual(metadata, wantMetadata) {
		t.Fatalf("expected metadata %#v, got %#v", wantMetadata, metadata)
	}
}

func TestResolveCaptureMetadataClipboard(t *testing.T) {
	original := readClipboard
	t.Cleanup(func() {
		readClipboard = original
	})

	s := &state.State{
		Workspace: &config.Workspace{
			Capture: config.CaptureConfig{
				Rules: []config.CaptureRule{
					{
						Action: config.CaptureAction{
							Clipboard: true,
							Tags:      []string{"clip"},
						},
					},
				},
			},
		},
	}

	readClipboard = func() (string, error) {
		return "", nil
	}

	tags, _, err := resolveCaptureMetadata(s, "", "")
	if err != nil {
		t.Fatalf("resolveCaptureMetadata returned error: %v", err)
	}
	if tags != nil {
		t.Fatalf("expected no tags when clipboard empty, got %v", tags)
	}

	readClipboard = func() (string, error) {
		return "hello", nil
	}

	tags, _, err = resolveCaptureMetadata(s, "", "")
	if err != nil {
		t.Fatalf("resolveCaptureMetadata returned error: %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"clip"}) {
		t.Fatalf("expected clipboard tags, got %v", tags)
	}

	readClipboard = func() (string, error) {
		return "", errors.New("boom")
	}

	if _, _, err := resolveCaptureMetadata(s, "", ""); err == nil {
		t.Fatalf("expected error when clipboard read fails")
	}
}
