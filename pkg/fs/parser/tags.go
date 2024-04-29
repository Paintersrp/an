package parser

import (
	"fmt"
	"os"
	"sort"

	tableTui "github.com/Paintersrp/an/tui/table"
	"github.com/Paintersrp/an/utils"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// TagHandler handles the logic related to tags.
type TagHandler struct {
	TagCounts map[string]int
	TagList   []string
}

// NewTagHandler creates a new instance of TagHandler.
func NewTagHandler() *TagHandler {
	return &TagHandler{
		TagCounts: make(map[string]int),
		TagList:   []string{},
	}
}

// ParseTag parses and counts a single tag.
func (th *TagHandler) ParseTag(tag string) {
	th.TagCounts[tag]++
	th.TagList = utils.AppendIfNotExists(th.TagList, tag)
}

// SortTagCounts sorts the tags by their counts.
func (th *TagHandler) SortTagCounts(order string) {
	// Convert map to a slice of key-value pairs.
	type kv struct {
		Key   string
		Value int
	}
	var tagCountsSlice []kv
	for k, v := range th.TagCounts {
		tagCountsSlice = append(tagCountsSlice, kv{k, v})
	}

	// Sort the slice based on the order parameter using a switch case.
	switch order {
	case "asc":
		sort.Slice(tagCountsSlice, func(i, j int) bool {
			return tagCountsSlice[i].Value < tagCountsSlice[j].Value
		})
	case "desc":
		sort.Slice(tagCountsSlice, func(i, j int) bool {
			return tagCountsSlice[i].Value > tagCountsSlice[j].Value
		})
	default:
		fmt.Println(
			"Invalid sort order. Use 'asc' for ascending or 'desc' for descending.",
		)
		return
	}

	// Update the TagList with sorted order.
	th.TagList = []string{}
	for _, kv := range tagCountsSlice {
		th.TagList = append(th.TagList, kv.Key)
	}
}

// PrintTagCounts prints the sorted list of tags and their counts.
func (th *TagHandler) PrintTagCounts() {
	fmt.Println("\nTag Counts:")
	for _, tag := range th.TagList {
		fmt.Printf("Tag: %s, Count: %d\n", tag, th.TagCounts[tag])
	}
}
func (th *TagHandler) PrintSortedTagCounts(order string) {
	th.SortTagCounts(order)
	fmt.Println("\nSorted Tag Counts:")
	for _, tag := range th.TagList {
		fmt.Printf("Tag: %s, Count: %d\n", tag, th.TagCounts[tag])
	}
}

func (th *TagHandler) setupTagTable() table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Tag", Width: 25},
		{Title: "Count", Width: 10},
	}

	var rows []table.Row
	th.SortTagCounts("desc")
	for id, tag := range th.TagList {
		rows = append(rows, []string{
			fmt.Sprintf("%d", id),
			tag,
			fmt.Sprintf("%d", th.TagCounts[tag]),
		})
	}

	tableCfg := tableTui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Focused: true,
		Height:  20,
	}

	t := tableCfg.ReturnTable()
	return t
}

func (th *TagHandler) ShowTagTable() {
	t := th.setupTagTable()
	m := tableTui.NewTableModel(t)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
