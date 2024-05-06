package notes

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"gopkg.in/yaml.v2"
)

func parseNoteFiles(noteFiles []string, vaultDir string, asFileDetails bool) []list.Item {
	var items []list.Item
	for _, fullPath := range noteFiles {
		fileWithoutVault := strings.TrimPrefix(
			fullPath,
			vaultDir+"/",
		)

		// Split the file path by the path separator
		pathParts := strings.Split(
			fileWithoutVault,
			string(filepath.Separator),
		)

		var fileName string

		if len(pathParts) < 2 {
			fileName = pathParts[0]
		} else {
			// The remaining parts joined together form the filename
			fileName = strings.Join(
				pathParts[1:],
				string(filepath.Separator),
			)
		}

		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		size := fileInfo.Size()
		lastModified := fileInfo.ModTime().Format(time.RFC1123)

		// Read the content of the note file
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		// Extract title and tags from front matter
		title, tags := parseFrontMatter(content, fileName)

		items = append(items, ListItem{
			fileName:     fileName,
			path:         fullPath,
			size:         size,
			lastModified: lastModified,
			title:        title,
			tags:         tags,
			showFullPath: asFileDetails,
		})
	}
	return items
}

// parseFrontMatter extracts title and tags from YAML front matter
func parseFrontMatter(
	content []byte,
	fileName string,
) (title string, tags []string) {
	// Get everything between the ---s
	re := regexp.MustCompile(`(?ms)^---\n(.+?)\n---`)
	match := re.FindSubmatch(content)
	if len(match) < 2 {
		return "", nil // no yaml content found
	}

	yamlContent := match[1]

	// Setup struct for binding the unmarshaled yamlContent
	var data struct {
		Title string   `yaml:"title"`
		Tags  []string `yaml:"tags"`
	}

	// Bind yamlContent to data struct, or give err
	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return fileName, nil // no data
	}

	// Return file name and tags
	return strings.TrimSpace(data.Title), data.Tags
}

func getSubdirectories(directory, excludeDir string) []string {
	files, err := os.ReadDir(directory)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	var subdirs []string
	for _, file := range files {
		if file.IsDir() && file.Name() != excludeDir {

			subdir := strings.TrimPrefix(filepath.Join(directory, file.Name()), directory)
			subdir = strings.TrimPrefix(
				subdir,
				string(os.PathSeparator),
			)
			subdirs = append(subdirs, subdir)
		}
	}
	return subdirs
}
