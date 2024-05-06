package notes

import (
	"fmt"
	"strings"
)

type ListItem struct {
	size         int64
	fileName     string
	path         string
	lastModified string
	title        string
	showFullPath bool
	tags         []string
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
	str := strings.Join(i.tags, " ")
	return fmt.Sprintf(
		"%s [%s]",
		i.title,
		str,
	)
}
