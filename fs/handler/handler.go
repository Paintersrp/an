package handler

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Paintersrp/an/fs/parser"
)

func WalkFiles(
	vaultDir string,
	excludeDirs []string,
	excludeFiles []string,
	modeFlag string,
) ([]string, error) {
	var files []string
	baseDepth := len(strings.Split(vaultDir, string(os.PathSeparator)))

	err := filepath.Walk(
		vaultDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err // exit
			}

			// Calculate the depth of the current path
			depth := len(strings.Split(path, string(os.PathSeparator)))

			// Skip files that are directly in the vaultDir
			if depth == baseDepth+1 && !info.IsDir() {
				return nil
			}

			// Check if the current directory is in the list of directories to exclude
			dir := filepath.Dir(path)
			for _, d := range excludeDirs {
				if dir == filepath.Join(vaultDir, d) {
					if info.IsDir() {
						return filepath.SkipDir // skip the entire directory
					}
					return nil // skip the single file
				}
			}

			// Check if the current file is in the list of files to exclude
			file := filepath.Base(path)
			for _, f := range excludeFiles {
				if file == f {
					return nil // skip this file
				}
			}

			// Skip hidden files or directories
			if strings.HasPrefix(file, ".") {
				if info.IsDir() {
					return filepath.SkipDir // skip directory if hidden
				}
				return nil // skip file if hidden
			}

			// Verify that the file has a .md extension (Markdown file)
			if !info.IsDir() && filepath.Ext(file) == ".md" {
				content, err := os.ReadFile(path)
				if err != nil {
					log.Printf("Error reading file: %s, error: %v", path, err)
					return nil // skip this file due to read error
				}

				switch modeFlag {
				case "orphan":
					// Only append the file if it does not contain note links
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

			return nil // walk on or finish
		},
	)

	// Return files and any errors
	return files, err
}
