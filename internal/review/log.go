package review

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Paintersrp/an/internal/templater"
)

var (
	filenameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9\-]+`)
	defaultLogDir     = "reviews"
)

var timestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

// LogMetadata captures the derived information about a persisted review log.
type LogMetadata struct {
	Path      string
	Filename  string
	Title     string
	Timestamp time.Time
	Preview   []string
}

// EnsureLogDir resolves and creates the directory used for persisted review logs.
// The configured path may be absolute or relative to the vault. When empty, the
// default "reviews" directory inside the vault is used. The resolved
// directory must live inside the vault.
func EnsureLogDir(vault, configured string) (string, string, error) {
	vault = strings.TrimSpace(vault)
	if vault == "" {
		return "", "", fmt.Errorf("vault directory is not configured")
	}

	dir := strings.TrimSpace(configured)
	if dir == "" {
		dir = filepath.Join(vault, defaultLogDir)
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(vault, dir)
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

// WriteMarkdownLog renders the checklist responses and resurfacing queue to a
// Markdown log file within the provided directory. The file name is derived
// from the manifest name and UTC timestamp. The written path is returned.
func WriteMarkdownLog(
	dir string,
	manifest templater.TemplateManifest,
	responses map[string]string,
	queue []ResurfaceItem,
	ts time.Time,
	vault string,
) (string, error) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	} else {
		ts = ts.UTC()
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	filename := buildReviewFilename(manifest, ts)
	path := filepath.Join(dir, filename+".md")
	content := renderReviewLogContent(manifest, responses, queue, ts, vault)
	if err := appendReviewLog(path, content); err != nil {
		return "", err
	}
	return path, nil
}

func buildReviewFilename(manifest templater.TemplateManifest, ts time.Time) string {
	return fmt.Sprintf("%s-%s", manifestSlug(manifest), ts.Format("2006-01-02"))
}

func manifestSlug(manifest templater.TemplateManifest) string {
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
	return name
}

func renderReviewLogContent(
	manifest templater.TemplateManifest,
	responses map[string]string,
	queue []ResurfaceItem,
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

// ListReviewLogs returns the metadata for review logs associated with the provided
// manifest and mode key. Results are sorted by the embedded timestamp in the log
// content when available (falling back to the file modification time) in
// descending order.
func ListReviewLogs(dir string, manifest templater.TemplateManifest, modeKey string) ([]LogMetadata, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	slug := strings.ToLower(manifestSlug(manifest))
	mode := strings.ToLower(strings.TrimSpace(modeKey))

	var logs []LogMetadata
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}

		base := strings.TrimSuffix(name, filepath.Ext(name))
		path := filepath.Join(dir, name)
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			if !matchesLogMode(base, "", slug, mode) {
				continue
			}
			logs = append(logs, LogMetadata{
				Path:      path,
				Filename:  name,
				Title:     defaultLogTitle(base),
				Timestamp: info.ModTime().UTC(),
			})
			continue
		}

		text := string(content)
		if !matchesLogMode(base, text, slug, mode) {
			continue
		}

		lines := normalizeLogLines(text)
		title := extractLogTitle(lines, base)
		preview := extractLogPreview(lines, 5)
		ts := extractLogTimestamp(lines, base, info.ModTime())

		logs = append(logs, LogMetadata{
			Path:      path,
			Filename:  name,
			Title:     title,
			Timestamp: ts,
			Preview:   preview,
		})
	}

	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].Timestamp.Equal(logs[j].Timestamp) {
			return strings.Compare(logs[i].Filename, logs[j].Filename) > 0
		}
		return logs[i].Timestamp.After(logs[j].Timestamp)
	})
	return logs, nil
}

func matchesLogMode(name, content, slug, mode string) bool {
	lowerName := strings.ToLower(name)
	if slug != "" && strings.Contains(lowerName, slug) {
		return true
	}
	if mode != "" && strings.Contains(lowerName, mode) {
		return true
	}

	if strings.TrimSpace(content) == "" {
		return slug == "" && mode == ""
	}

	lines := normalizeLogLines(content)
	title := strings.ToLower(extractLogTitle(lines, name))
	if slug != "" && strings.Contains(title, slug) {
		return true
	}
	if mode != "" && strings.Contains(title, mode) {
		return true
	}
	return false
}

func extractLogTitle(lines []string, fallback string) string {
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if i == 0 && strings.HasPrefix(line, "---") {
			i = skipFrontMatter(lines, i)
			continue
		}
		if strings.HasPrefix(line, "#") {
			candidate := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if idx := strings.Index(candidate, "—"); idx >= 0 {
				candidate = strings.TrimSpace(candidate[:idx])
			}
			if candidate != "" {
				return candidate
			}
		}
		if !strings.HasPrefix(line, "#") {
			break
		}
	}
	return defaultLogTitle(fallback)
}

func extractLogPreview(lines []string, limit int) []string {
	if limit <= 0 {
		limit = 5
	}
	var preview []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "---") && (i == 0) {
			i = skipFrontMatter(lines, i)
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		value := strings.TrimRight(line, " \t")
		value = strings.TrimLeft(value, "\t")
		preview = append(preview, strings.TrimSpace(value))
		if len(preview) >= limit {
			break
		}
	}
	return preview
}

func extractLogTimestamp(lines []string, fallback string, mod time.Time) time.Time {
	for _, line := range lines {
		if match := timestampPattern.FindString(line); match != "" {
			if ts, err := time.Parse(time.RFC3339, match); err == nil {
				return ts.UTC()
			}
		}
	}

	if date := findDateSuffix(fallback); !date.IsZero() {
		return date
	}
	return mod.UTC()
}

func skipFrontMatter(lines []string, index int) int {
	for i := index + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i
		}
	}
	return len(lines)
}

func normalizeLogLines(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

func defaultLogTitle(value string) string {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.TrimSuffix(cleaned, ".md")
	cleaned = strings.ReplaceAll(cleaned, "_", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "Review"
	}
	parts := strings.Fields(cleaned)
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		lower := strings.ToLower(part)
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, " ")
}

func findDateSuffix(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < len("2006-01-02") {
		return time.Time{}
	}
	tail := trimmed[len(trimmed)-len("2006-01-02"):]
	if ts, err := time.Parse("2006-01-02", tail); err == nil {
		return ts
	}
	return time.Time{}
}
