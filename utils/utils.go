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

func RenderMarkdownPreview(
	path string,
	w, h int,
) string {
	const cutoff = 1000

	content, err := os.ReadFile(path)
	if err != nil {
		return "Error reading file"
	}

	// Check if the content exceeds the cutoff and trim if necessary
	if len(content) > cutoff {
		content = content[:cutoff]
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(100),
		glamour.WithColorProfile(termenv.ANSI256),
	)

	markdown, err := r.Render(string(content))
	if err != nil {
		return "Error rendering markdown"
	}

	return markdown
}

func FormatBytes(size int64) string {
	var units = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	var mod int64 = 1024
	var i int
	for size >= mod {
		size /= mod
		i++
	}
	return fmt.Sprintf("%d %s", size, units[i])
}
