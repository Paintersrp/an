package handler

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Paintersrp/an/internal/parser"
)

type FileHandler struct {
	vaultDir string
}

func NewFileHandler(vaultDir string) *FileHandler {
	return &FileHandler{vaultDir: vaultDir}
}

// Archive moves a note file to the archive subdirectory.
func (h *FileHandler) Archive(path string) error {
	subDir, err := filepath.Rel(h.vaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	archiveSubDir := filepath.Join(h.vaultDir, "archive", subDir)
	if err := os.MkdirAll(archiveSubDir, os.ModePerm); err != nil {
		return err
	}

	newPath := filepath.Join(archiveSubDir, filepath.Base(path))
	return os.Rename(path, newPath)
}

// Unarchive moves a note file from the archive subdirectory to its original location.
func (h *FileHandler) Unarchive(path string) error {
	subDir, err := filepath.Rel(filepath.Join(h.vaultDir, "archive"), filepath.Dir(path))
	if err != nil {
		return err
	}

	originalDir := filepath.Join(h.vaultDir, subDir)
	newPath := filepath.Join(originalDir, filepath.Base(path))
	return os.Rename(path, newPath)
}

// Trash moves a note file to the trash subdirectory.
func (h *FileHandler) Trash(path string) error {
	subDir, err := filepath.Rel(h.vaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	trashDir := filepath.Join(h.vaultDir, "trash", subDir)
	if err := os.MkdirAll(trashDir, os.ModePerm); err != nil {
		return err
	}

	newPath := filepath.Join(trashDir, filepath.Base(path))
	return os.Rename(path, newPath)
}

// Untrash moves a note file from the trash subdirectory to its original location.
func (h *FileHandler) Untrash(path string) error {
	subDir, err := filepath.Rel(filepath.Join(h.vaultDir, "trash"), filepath.Dir(path))
	if err != nil {
		return err
	}

	originalDir := filepath.Join(h.vaultDir, subDir)
	newPath := filepath.Join(originalDir, filepath.Base(path))
	return os.Rename(path, newPath)
}

func (h *FileHandler) WalkFiles(
	excludeDirs []string,
	excludeFiles []string,
	modeFlag string,
) ([]string, error) {
	var files []string
	baseDepth := len(strings.Split(h.vaultDir, string(os.PathSeparator)))

	err := filepath.Walk(
		h.vaultDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Calculate the depth of the current path
			depth := len(strings.Split(path, string(os.PathSeparator)))

			// Skip files that are directly in the vaultDir
			if depth == baseDepth+1 && !info.IsDir() {
				return nil
			}

			// Check if the current directory is in the excluded list
			dir := filepath.Dir(path)
			for _, d := range excludeDirs {
				if dir == filepath.Join(h.vaultDir, d) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			// Check if the current file is in the list of files to exclude
			file := filepath.Base(path)
			for _, f := range excludeFiles {
				if file == f {
					return nil
				}
			}

			// Skip hidden files or directories
			if strings.HasPrefix(file, ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Verify that the file has a .md extension (Markdown file)
			if !info.IsDir() && filepath.Ext(file) == ".md" {
				content, err := os.ReadFile(path)
				if err != nil {
					log.Printf("Error reading file: %s, error: %v", path, err)
					return nil
				}

				switch modeFlag {
				case "orphan":
					// Append the file if it does not contain note links
					if !parser.HasNoteLinks(content) {
						files = append(files, path)
					}
				case "unfulfilled":
					if parser.CheckFulfillment(content, "false") {
						files = append(files, path)
					}
				default:
					files = append(files, path)
				}
			}

			return nil
		},
	)

	return files, err
}

func (h *FileHandler) GetSubdirectories(directory, excludeDir string) []string {
	files, err := os.ReadDir(directory)
	if err != nil {
		// Should probably properly propagate this error back up the application
		log.Fatalf("Failed to read directory: %v", err)
	}

	var subDirs []string
	for _, f := range files {
		if f.IsDir() && f.Name() != excludeDir {

			subDir := strings.TrimPrefix(filepath.Join(directory, f.Name()), directory)
			subDir = strings.TrimPrefix(
				subDir,
				string(os.PathSeparator),
			)
			subDirs = append(subDirs, subDir)
		}
	}
	return subDirs
}
