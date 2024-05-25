package sync

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/yaml.v1"
)

func getMarkdownFiles(tree *object.Tree) ([]string, error) {
	var files []string

	treeWalker := object.NewTreeWalker(tree, true, nil)
	defer treeWalker.Close()

	for {
		name, entry, err := treeWalker.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !entry.Mode.IsFile() {
			continue
		}

		if filepath.Ext(name) == ".md" {
			files = append(files, name)
		}
	}

	return files, nil
}

func splitFrontMatter(note string) (frontMatter, content string) {
	parts := strings.Split(note, "\n---\n")
	if len(parts) < 2 {
		return "", note
	}

	return strings.TrimSpace(parts[0]), parts[1]
}

func parseFrontMatter(frontMatter string) (Frontmatter, error) {
	var metadata Frontmatter
	err := yaml.Unmarshal([]byte(frontMatter), &metadata)
	if err != nil {
		return Frontmatter{}, fmt.Errorf("error parsing front matter: %v", err)
	}

	return metadata, nil
}

func parseLinkedNotes(content string) []string {
	pattern := regexp.MustCompile(`\[\[(.*?)\]\]`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	linkedNotes := make([]string, 0, len(matches))

	for _, match := range matches {
		linkedNotes = append(linkedNotes, match[1])
	}

	return linkedNotes
}
