package review

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Paintersrp/an/internal/templater"
)

func TestRunChecklistCollectsResponsesAndSuggestions(t *testing.T) {
	t.Parallel()

	manifest := templater.TemplateManifest{
		Name: "review-daily",
		Fields: []templater.TemplateField{
			{Key: "clear-inbox", Label: "Clear inbox", Prompt: "Triage the inbox", Defaults: []string{"inbox"}},
			{Key: "plan", Prompt: "Plan"},
		},
	}

	queue := []ResurfaceItem{
		{
			Path:       "notes/inbox.md",
			Tags:       []string{"inbox"},
			ModifiedAt: time.Now().Add(-72 * time.Hour),
			Age:        72 * time.Hour,
			Bucket:     "every-3-days",
		},
	}

	input := strings.NewReader("done\nnext\n")
	var output bytes.Buffer

	responses, err := RunChecklist(manifest, queue, input, &output)
	if err != nil {
		t.Fatalf("RunChecklist returned error: %v", err)
	}

	if responses["clear-inbox"] != "done" || responses["plan"] != "next" {
		t.Fatalf("unexpected responses: %#v", responses)
	}

	text := output.String()
	if !strings.Contains(text, "Related resurfacing candidates") {
		t.Fatalf("expected suggestions in output, got %q", text)
	}
}
