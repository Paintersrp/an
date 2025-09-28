package review

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Paintersrp/an/internal/note"
	reviewsvc "github.com/Paintersrp/an/internal/review"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

var (
	openReviewNote    = note.OpenFromPath
	filenameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9\-]+`)
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

	dir, subdir, err := ensureReviewDir(st)
	if err != nil {
		return "", err
	}

	filename := buildReviewFilename(manifest, ts)
	noteRef := note.NewZettelkastenNote(vault, subdir, filename, nil, nil, "")
	if _, err := noteRef.EnsurePath(); err != nil {
		return "", err
	}

	path := filepath.Join(dir, filename+".md")
	content := renderReviewLogContent(manifest, responses, queue, ts, vault)
	if err := appendReviewLog(path, content); err != nil {
		return "", err
	}

	if err := openReviewNote(path, false); err != nil {
		return "", err
	}

	return path, nil
}

func ensureReviewDir(st *state.State) (string, string, error) {
	vault := strings.TrimSpace(st.Vault)
	dir := ""
	if st.Workspace != nil {
		if path := strings.TrimSpace(st.Workspace.NamedPins["review"]); path != "" {
			if filepath.IsAbs(path) {
				dir = path
			} else {
				dir = filepath.Join(vault, path)
			}
		}
	}
	if dir == "" {
		dir = filepath.Join(vault, ".an", "review")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}

	rel, err := filepath.Rel(vault, dir)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", "", fmt.Errorf("review directory %q must be inside the vault", dir)
	}
	rel = filepath.ToSlash(rel)
	return dir, rel, nil
}

func buildReviewFilename(manifest templater.TemplateManifest, ts time.Time) string {
	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		name = "review"
	}
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = filenameSanitizer.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		name = "review"
	}
	return fmt.Sprintf("%s-%s", name, ts.Format("2006-01-02"))
}

func renderReviewLogContent(
	manifest templater.TemplateManifest,
	responses map[string]string,
	queue []reviewsvc.ResurfaceItem,
	ts time.Time,
	vault string,
) string {
	var builder strings.Builder

	title := strings.TrimSpace(manifest.Name)
	if title == "" {
		title = "Review"
	}
	fmt.Fprintf(&builder, "## %s — %s UTC\n\n", title, ts.Format(time.RFC3339))

	if desc := strings.TrimSpace(manifest.Description); desc != "" {
		builder.WriteString(desc)
		builder.WriteString("\n\n")
	}

	builder.WriteString("### Checklist responses\n\n")
	if len(manifest.Fields) == 0 {
		builder.WriteString("- _No checklist steps configured._\n")
	} else {
		for _, field := range manifest.Fields {
			label := strings.TrimSpace(field.Label)
			if label == "" {
				label = humanizeKey(field.Key)
			}
			response := strings.TrimSpace(responses[field.Key])
			if response == "" {
				fmt.Fprintf(&builder, "- **%s:** _(no response)_\n", label)
				continue
			}
			if strings.Contains(response, "\n") {
				fmt.Fprintf(&builder, "- **%s:**\n", label)
				lines := strings.Split(response, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) == "" {
						builder.WriteString("  \n")
					} else {
						fmt.Fprintf(&builder, "  %s\n", line)
					}
				}
			} else {
				fmt.Fprintf(&builder, "- **%s:** %s\n", label, response)
			}
		}
	}

	builder.WriteString("\n### Resurfacing queue\n\n")
	if len(queue) == 0 {
		builder.WriteString("- _No resurfacing candidates._\n")
	} else {
		for _, item := range queue {
			path := item.Path
			if rel, err := filepath.Rel(vault, item.Path); err == nil && !strings.HasPrefix(rel, "..") {
				path = filepath.ToSlash(rel)
			}
			last := item.ModifiedAt.UTC().Format("2006-01-02")
			bucket := strings.TrimSpace(item.Bucket)
			if bucket == "" {
				bucket = "unscheduled"
			}
			fmt.Fprintf(&builder, "- %s — last touched %s (%s)\n", path, last, bucket)
		}
	}

	builder.WriteString("\n")
	return builder.String()
}

func appendReviewLog(path, content string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	entry := strings.TrimRight(content, "\n") + "\n"
	if info.Size() > 0 {
		entry = "\n\n" + entry
	}

	_, err = file.WriteString(entry)
	return err
}
