package notes

import (
	"fmt"
	"strings"

	"github.com/Paintersrp/an/internal/cache"
)

type ListItem struct {
	fileName     string
	path         string
	lastModified string
	title        string
	subdirectory string
	tags         []string
	size         int64
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
	description := ""

	if i.showFullPath {
		description += fmt.Sprintf(
			"Size: %s, Last Modified: %s",
			cache.ReadableSize(i.size),
			i.lastModified,
		)
	} else {
		if i.subdirectory != "" {
			description += fmt.Sprintf("[%s] ", i.subdirectory)
		}

		if len(i.tags) == 0 {
			description += "No tags"
		} else {
			description += strings.Join(i.tags, ", ")
		}
	}
	return description
}

func (i ListItem) FilterValue() string {
        str := strings.Join(i.tags, " ")
        return fmt.Sprintf(
                "%s [%s] [%s]",
                i.Title(),
                str,
                i.subdirectory,
        )
}

func (i ListItem) Path() string {
	return i.path
}
