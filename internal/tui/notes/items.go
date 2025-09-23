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
	highlights   *highlightStore
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
	if snippet := i.highlightSnippet(); snippet != "" {
		if description != "" {
			description += "\n"
		}
		description += snippet
	}

	return description
}

func (i ListItem) FilterValue() string {
	str := strings.Join(i.tags, " ")
	parts := []string{i.Title(), "[" + str + "]", "[" + i.subdirectory + "]"}
	if snippet := i.highlightSnippet(); snippet != "" {
		parts = append(parts, snippet)
	}
	return strings.Join(parts, " ")
}

func (i ListItem) Path() string {
	return i.path
}

func (i ListItem) highlightSnippet() string {
	if i.highlights == nil {
		return ""
	}
	if res, ok := i.highlights.lookup(i.path); ok {
		if res.Snippet != "" {
			return res.Snippet
		}
		return res.MatchFrom
	}
	return ""
}
