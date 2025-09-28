package review

import (
	"errors"
	"strings"
	"time"

	"github.com/Paintersrp/an/internal/note"
	reviewsvc "github.com/Paintersrp/an/internal/review"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

var (
	openReviewNote = note.OpenFromPath
)

func persistReviewLog(
	st *state.State,
	manifest templater.TemplateManifest,
	responses map[string]string,
	queue []reviewsvc.ResurfaceItem,
	ts time.Time,
) (string, error) {
	if st == nil {
		return "", errors.New("state is not configured")
	}

	vault := strings.TrimSpace(st.Vault)
	if vault == "" {
		return "", errors.New("vault directory is not configured")
	}

	if ts.IsZero() {
		ts = time.Now().UTC()
	} else {
		ts = ts.UTC()
	}

	dir, _, err := ensureReviewDir(st)
	if err != nil {
		return "", err
	}

	path, err := reviewsvc.WriteMarkdownLog(dir, manifest, responses, queue, ts, vault)
	if err != nil {
		return "", err
	}

	if err := openReviewNote(path, false); err != nil {
		return "", err
	}

	return path, nil
}

func ensureReviewDir(st *state.State) (string, string, error) {
	vault := strings.TrimSpace(st.Vault)
	configured := ""
	if st.Workspace != nil {
		configured = st.Workspace.NamedPins["review"]
	}
	return reviewsvc.EnsureLogDir(vault, configured)
}
