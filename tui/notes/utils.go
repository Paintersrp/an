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
	for _, p := range noteFiles {
		fileWithoutVault := strings.TrimPrefix(
			p,
			vaultDir+"/",
		)

		// Split the file path by the path separator
		parts := strings.Split(
			fileWithoutVault,
			string(filepath.Separator),
		)

		var n string

		if len(parts) < 2 {
			n = parts[0]
		} else {
			// The remaining parts joined together form the filename
			n = strings.Join(
				parts[1:],
				string(filepath.Separator),
			)
		}

		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		size := info.Size()
		last := info.ModTime().Format(time.RFC1123)

		// Read the content of the note file
		c, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		// Extract title and tags from front matter
		title, tags := parseFrontMatter(c, n)

		items = append(items, ListItem{
			fileName:     n,
			path:         p,
			size:         size,
			lastModified: last,
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
	m := re.FindSubmatch(content)
	if len(m) < 2 {
		return "", nil // no yaml content found
	}

	yamlContent := m[1]

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
