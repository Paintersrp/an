package parser

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	metadataPattern  = regexp.MustCompile(`@([a-zA-Z0-9_-]+)\(([^)]+)\)`)
	backlinkPattern  = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	supportedDateFmt = []string{time.RFC3339, "2006-01-02", "2006/01/02"}
)

// ExtractTaskMetadata parses inline metadata tokens from a Markdown task body. It returns the
// cleaned content along with a populated TaskMetadata struct. Metadata tokens follow the
// @key(value) syntax. Supported keys include due, scheduled, priority, owner, assignee, and project.
func ExtractTaskMetadata(content string) (string, TaskMetadata) {
	metadata := TaskMetadata{RawTokens: make(map[string]string)}
	trimmed := strings.TrimSpace(content)

	cleaned := metadataPattern.ReplaceAllStringFunc(trimmed, func(match string) string {
		sub := metadataPattern.FindStringSubmatch(match)
		if len(sub) < 3 {
			return ""
		}

		key := strings.ToLower(strings.TrimSpace(sub[1]))
		value := strings.TrimSpace(sub[2])
		if value == "" {
			return ""
		}

		metadata.RawTokens[key] = value

		switch key {
		case "due":
			if t, ok := parseDate(value); ok {
				metadata.DueDate = &t
			}
		case "scheduled", "schedule", "start":
			if t, ok := parseDate(value); ok {
				metadata.ScheduledDate = &t
			}
		case "priority":
			metadata.Priority = strings.ToLower(value)
		case "owner", "assignee", "responsible":
			metadata.Owner = value
		case "project", "group":
			metadata.Project = value
		}

		return ""
	})

	refs := backlinkPattern.FindAllStringSubmatch(trimmed, -1)
	if len(refs) > 0 {
		metadata.References = make([]string, 0, len(refs))
		seen := make(map[string]struct{})
		for _, r := range refs {
			if len(r) < 2 {
				continue
			}
			ref := strings.TrimSpace(r[1])
			if ref == "" {
				continue
			}
			if _, exists := seen[ref]; exists {
				continue
			}
			seen[ref] = struct{}{}
			metadata.References = append(metadata.References, ref)
		}
		sort.Strings(metadata.References)
	}

	cleaned = backlinkPattern.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned, metadata
}

func parseDate(value string) (time.Time, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return time.Time{}, false
	}

	// Support relative shortcuts for today and tomorrow.
	lower := strings.ToLower(normalized)
	switch lower {
	case "today":
		now := time.Now().Local()
		t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return t, true
	case "tomorrow":
		now := time.Now().Local().Add(24 * time.Hour)
		t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return t, true
	}

	for _, layout := range supportedDateFmt {
		if parsed, err := time.Parse(layout, normalized); err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}
