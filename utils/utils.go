package utils

import (
	"fmt"
	"os"
	"path/filepath"
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

// GenerateDate generates a date string based on the given type (day, week, month, year).
func GenerateDate(numUnits int, unitType string) string {
	var date time.Time
	var dateFormat string
	now := time.Now()

	switch unitType {
	case "day":
		date = now.AddDate(0, 0, numUnits)
		dateFormat = "20060102"
	case "week":
		// Find Sunday of the current week
		offset := int(time.Sunday - now.Weekday())
		if offset > 0 {
			offset = -6
		}
		startOfWeek := now.AddDate(0, 0, offset)
		// Add the number of weeks
		date = startOfWeek.AddDate(0, 0, numUnits*7)
		dateFormat = "20060102"
	case "month":
		// Find the first day of the current month
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		// Add the number of months
		date = startOfMonth.AddDate(0, numUnits, 0)
		dateFormat = "200601"
	case "year":
		// Find the first day of the current year
		startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		// Add the number of years
		date = startOfYear.AddDate(numUnits, 0, 0)
		dateFormat = "2006"
	default:
		// Default to today's date
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

	// Read file from system
	content, err := os.ReadFile(path)
	if err != nil {
		return "Error reading file"
	}

	// Check if the content exceeds the cutoff and trim if necessary
	if len(content) > cutoff {
		content = content[:cutoff]
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

func Archive(path string, vaultDir string) error {
	// Get the subdirectory path relative to the vault directory
	subDir, err := filepath.Rel(vaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	// Create the archive subdirectory path, if needed
	archiveSubDir := filepath.Join(vaultDir, "archive", subDir)
	if _, err := os.Stat(archiveSubDir); os.IsNotExist(err) {
		if err := os.MkdirAll(archiveSubDir, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the note to the archive subdirectory
	newPath := filepath.Join(archiveSubDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

func Unarchive(path string, vaultDir string) error {
	// Infer the original subdirectory from the archive path
	subDir, err := filepath.Rel(
		filepath.Join(vaultDir, "archive"),
		filepath.Dir(path),
	)
	if err != nil {
		return err
	}

	// Define the original directory where the notes should be restored
	originalDir := filepath.Join(vaultDir, subDir)

	// Move the note from the archive directory back to the original directory
	newPath := filepath.Join(originalDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

// Function to move a note to the trash directory
func Trash(path string, vaultDir string) error {
	// Get the subdirectory path relative to the vault directory
	subDir, err := filepath.Rel(vaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	// Define the trash directory path
	trashDir := filepath.Join(vaultDir, "trash", subDir)
	if _, err := os.Stat(trashDir); os.IsNotExist(err) {
		if err := os.MkdirAll(trashDir, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the note to the trash directory
	newPath := filepath.Join(trashDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

// Function to restore a note from the trash directory
func Untrash(path string, vaultDir string) error {
	// Infer the original subdirectory from the archive path
	subDir, err := filepath.Rel(
		filepath.Join(vaultDir, "trash"),
		filepath.Dir(path),
	)
	if err != nil {
		return err
	}

	// Define the original directory where the notes should be restored
	originalDir := filepath.Join(vaultDir, subDir)

	// Move the note from the trash directory back to the original directory
	newPath := filepath.Join(originalDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

// converts bytes to a human-readable format.
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
