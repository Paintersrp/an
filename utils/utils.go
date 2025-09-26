package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
)

func AppendIfNotExists(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

func ValidateInput(input string) ([]string, error) {
	if input == "" {
		return []string{}, nil
	}

	items := strings.Split(input, " ")
	for _, item := range items {
		if !isValidInput(item) {
			return nil, fmt.Errorf(
				"invalid input '%s': Input must only contain alphanumeric characters, hyphens, and underscores",
				item,
			)
		}
	}
	return items, nil
}

func isValidInput(input string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(input)
}

func GenerateDate(numUnits int, unitType string) string {
	var date time.Time
	var dateFormat string
	now := time.Now()

	switch unitType {
	case "day":
		date = now.AddDate(0, 0, numUnits)
		dateFormat = "20060102"
	case "week":
		offset := int(time.Sunday - now.Weekday())
		if offset > 0 {
			offset = -6
		}
		startOfWeek := now.AddDate(0, 0, offset)
		date = startOfWeek.AddDate(0, 0, numUnits*7)
		dateFormat = "20060102"
	case "month":
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		date = startOfMonth.AddDate(0, numUnits, 0)
		dateFormat = "200601"
	case "year":
		startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		date = startOfYear.AddDate(numUnits, 0, 0)
		dateFormat = "2006"
	default:
		date = now
		dateFormat = "20060102"
	}

	return date.Format(dateFormat)
}

func ReadFileAndTrimContent(path string, cutoff int) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if len(content) > cutoff {
		content = content[:cutoff]
	}

	return string(content), nil
}

func ParseFrontmatter(content string) (string, string) {
	frontmatterRegex := regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n?`)
	matches := frontmatterRegex.FindStringSubmatch(content)

	var frontmatter, markdown string
	if len(matches) > 1 {
		frontmatter = matches[1]
		markdown = strings.TrimPrefix(content, matches[0])
	} else {
		markdown = content
	}

	return frontmatter, markdown
}

func FormatFrontmatterAsMarkdown(frontmatter string) string {
	lines := strings.Split(frontmatter, "\n")
	formattedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if value != "" {
				formattedLines = append(
					formattedLines,
					fmt.Sprintf("**%s:** %s", key, value),
				)
			}
		} else if line != "" {
			formattedLines = append(formattedLines, line)
		}
	}

	return strings.Join(formattedLines, "\n\n")
}

func RenderMarkdownPreview(path string, w, h int, cutoff int) (string, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "Error reading file", false
	}

	trimmed := false
	if cutoff > 0 && len(content) > cutoff {
		content = content[:cutoff]
		trimmed = true
	}

	frontmatter, markdown := ParseFrontmatter(string(content))
	formattedFrontmatter := FormatFrontmatterAsMarkdown(frontmatter)

	var renderedContent string
	if formattedFrontmatter != "" {
		renderedContent = formattedFrontmatter + "\n\n---\n\n\n" + markdown
	} else {
		renderedContent = "No frontmatter found.\n\n---\n\n\n" + markdown
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(100),
		glamour.WithColorProfile(termenv.ANSI256),
	)

	renderedMarkdown, err := r.Render(renderedContent)
	if err != nil {
		return "Error rendering markdown", trimmed
	}

	return renderedMarkdown, trimmed
}

func FormatBytes(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	var mod int64 = 1024
	var i int
	for size >= mod {
		size /= mod
		i++
	}
	return fmt.Sprintf("%d %s", size, units[i])
}
