package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Paintersrp/an/internal/config"
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

func Archive(path string, cfg *config.Config) error {
	// Get the subdirectory path relative to the vault directory
	subDir, err := filepath.Rel(cfg.VaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	// Create the archive subdirectory path, if needed
	archiveSubDir := filepath.Join(cfg.VaultDir, "archive", subDir)
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

func Unarchive(path string, cfg *config.Config) error {
	// Infer the original subdirectory from the archive path
	subDir, err := filepath.Rel(
		filepath.Join(cfg.VaultDir, "archive"),
		filepath.Dir(path),
	)
	if err != nil {
		return err
	}

	// Define the original directory where the notes should be restored
	originalDir := filepath.Join(cfg.VaultDir, subDir)

	// Move the note from the archive directory back to the original directory
	newPath := filepath.Join(originalDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

// Function to move a note to the trash directory
func Trash(path string, cfg *config.Config) error {
	// Get the subdirectory path relative to the vault directory
	subDir, err := filepath.Rel(cfg.VaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	// Define the trash directory path
	trashDir := filepath.Join(cfg.VaultDir, "trash", subDir)
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
func Untrash(path string, cfg *config.Config) error {
	// Infer the original subdirectory from the archive path
	subDir, err := filepath.Rel(
		filepath.Join(cfg.VaultDir, "trash"),
		filepath.Dir(path),
	)
	if err != nil {
		return err
	}

	// Define the original directory where the notes should be restored
	originalDir := filepath.Join(cfg.VaultDir, subDir)

	// Move the note from the trash directory back to the original directory
	newPath := filepath.Join(originalDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}
