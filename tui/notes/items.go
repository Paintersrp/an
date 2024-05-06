package notes

import (
	"fmt"
	"strings"
)

type ListItem struct {
	fileName     string
	path         string
	size         int64
	lastModified string
	title        string
	tags         []string
	showFullPath bool
}

func (i ListItem) Title() string {
	if i.showFullPath {
		return i.path
	}
	if i.title == "" {
		return strings.TrimSuffix(i.fileName, ".md")
	}
	return i.title
}

func (i ListItem) Description() string {
	if i.showFullPath {
		return fmt.Sprintf("Size: %d bytes, Last Modified: %s", i.size, i.lastModified)
	}
	if len(i.tags) == 0 {
		return "No tags"
	}
	return strings.Join(i.tags, ", ")
}

func (i ListItem) FilterValue() string {
	tagString := strings.Join(i.tags, " ") // Join all tags with a space
	return fmt.Sprintf(
		"%s [%s]",
		i.title,
		tagString,
	) // Include tags in square brackets for clarity
}
