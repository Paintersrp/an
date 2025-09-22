package notes

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
)

type sortField int

const (
	sortByTitle sortField = iota
	sortBySubdir
	sortByModifiedAt
)

type sortOrder int

const (
	ascending sortOrder = iota
	descending
)

func sortItems(items []ListItem, field sortField, order sortOrder) []list.Item {
	sortedItems := make([]ListItem, len(items))
	copy(sortedItems, items)

	sort.Slice(sortedItems, func(i, j int) bool {
		switch field {
		case sortByTitle:
			iTitle := titleForSort(sortedItems[i])
			jTitle := titleForSort(sortedItems[j])
			if order == ascending {
				return strings.Compare(iTitle, jTitle) < 0
			}
			return strings.Compare(iTitle, jTitle) > 0
		case sortBySubdir:
			if order == ascending {
				return strings.Compare(
					sortedItems[i].subdirectory,
					sortedItems[j].subdirectory,
				) < 0
			}
			return strings.Compare(
				sortedItems[i].subdirectory,
				sortedItems[j].subdirectory,
			) > 0
		case sortByModifiedAt:
			iTime := parseDate(sortedItems[i].lastModified)
			jTime := parseDate(sortedItems[j].lastModified)
			if order == ascending {
				return iTime.Before(jTime)
			}
			return iTime.After(jTime)
		default:
			// Handle default case
		}
		return false
	})

	listItems := make([]list.Item, len(sortedItems))
	for i, item := range sortedItems {
		listItems[i] = item
	}

	return listItems
}

func parseDate(dateStr string) time.Time {
	layout := "Mon, 02 Jan 2006 15:04:05 MST"
	t, _ := time.Parse(layout, dateStr)
	return t
}

func titleForSort(item ListItem) string {
	if item.title == "" {
		return strings.TrimSuffix(item.fileName, ".md")
	}
	return item.title
}
