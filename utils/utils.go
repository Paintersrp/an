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
		return []string{}, nil // No input provided, return an empty slice
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
	// Define the criteria for a valid input, for example:
	// A valid input contains only letters, numbers, hyphens, and underscores.
	return regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(input)
}

func IncrementDays(numDays int) string {
	return time.Now().AddDate(0, 0, numDays).Format("20060102")
}

func RenderMarkdownPreview(
	path string,
	w, h int,
) string {
	// Read file from system
	content, err := os.ReadFile(path)
	if err != nil {
		return "Error reading file"
	}

	// Initiate glamour renderer to add colors to our markdown preview
	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(100),
		glamour.WithColorProfile(termenv.ANSI256),
	)

	// Render formatted and styled markdown content
	markdown, err := r.Render(string(content))
	if err != nil {
		return "Error rendering markdown" // Displayed in Preview Pane
	}

	// Return markdown output
	return markdown
}
