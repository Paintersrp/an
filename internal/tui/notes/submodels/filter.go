package submodels

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FilterSelectionChangedMsg struct {
	Tags     []string
	Metadata map[string][]string
}

type FilterClosedMsg struct{}

type filterOptionKind int

const (
	filterOptionHeader filterOptionKind = iota
	filterOptionEntry
	filterOptionEmpty
)

type filterOption struct {
	kind       filterOptionKind
	label      string
	key        string
	value      string
	selectable bool
}

type FilterModel struct {
	cursor            int
	options           []filterOption
	tags              []string
	metadata          map[string][]string
	selectedTags      map[string]struct{}
	selectedMetadata  map[string]map[string]struct{}
	hasSelectableOpts bool
}

var (
	filterTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0AF"))
	filterHeaderStyle   = lipgloss.NewStyle().Bold(true)
	filterCursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#0AF"))
	filterInactiveStyle = lipgloss.NewStyle()
	filterHelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#94e2d5"))
	filterEmptyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))
)

func NewFilterModel() *FilterModel {
	return &FilterModel{
		selectedTags:     make(map[string]struct{}),
		selectedMetadata: make(map[string]map[string]struct{}),
	}
}

func (m *FilterModel) SetOptions(tags []string, metadata map[string][]string) {
	m.tags = append([]string(nil), tags...)
	sort.Strings(m.tags)

	m.metadata = make(map[string][]string, len(metadata))
	for key, values := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || ShouldExcludeMetadataKey(trimmedKey) {
			continue
		}

		filteredValues := make([]string, 0, len(values))
		for _, value := range values {
			trimmedValue := strings.TrimSpace(value)
			if trimmedValue == "" {
				continue
			}
			filteredValues = append(filteredValues, trimmedValue)
		}
		if len(filteredValues) == 0 {
			continue
		}

		sort.Strings(filteredValues)
		m.metadata[trimmedKey] = filteredValues
	}

	m.rebuildOptions()
}

func (m *FilterModel) SetSelection(tags []string, metadata map[string][]string) {
	m.selectedTags = make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		normalized := strings.TrimSpace(tag)
		if normalized == "" {
			continue
		}
		m.selectedTags[normalized] = struct{}{}
	}

	m.selectedMetadata = make(map[string]map[string]struct{}, len(metadata))
	for key, values := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if _, ok := m.selectedMetadata[trimmedKey]; !ok {
			m.selectedMetadata[trimmedKey] = make(map[string]struct{})
		}
		for _, value := range values {
			normalized := strings.TrimSpace(value)
			if normalized == "" {
				continue
			}
			m.selectedMetadata[trimmedKey][normalized] = struct{}{}
		}
	}

	m.ensureCursor()
}

func (m *FilterModel) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type { //nolint:exhaustive // handled via default
		case tea.KeyUp:
			fallthrough
		case tea.KeyCtrlP:
			m.moveCursor(-1)
			return nil, true
		case tea.KeyDown:
			fallthrough
		case tea.KeyCtrlN:
			m.moveCursor(1)
			return nil, true
		case tea.KeySpace:
			if m.toggleCurrent() {
				return m.selectionChangedCmd(), true
			}
			return nil, true
		case tea.KeyCtrlL:
			m.clearSelections()
			return m.selectionChangedCmd(), true
		case tea.KeyEnter:
			fallthrough
		case tea.KeyEsc:
			fallthrough
		case tea.KeyCtrlC:
			return func() tea.Msg { return FilterClosedMsg{} }, true
		}

		switch msg.String() {
		case "j":
			m.moveCursor(1)
			return nil, true
		case "k":
			m.moveCursor(-1)
			return nil, true
		}
	}

	return nil, false
}

func (m *FilterModel) View() string {
	var lines []string
	lines = append(lines, filterTitleStyle.Render("Filter notes"))

	for idx, opt := range m.options {
		switch opt.kind {
		case filterOptionHeader:
			lines = append(lines, "", filterHeaderStyle.Render(opt.label))
		case filterOptionEmpty:
			lines = append(lines, filterEmptyStyle.Render(opt.label))
		case filterOptionEntry:
			indicator := "[ ]"
			if m.isSelected(opt) {
				indicator = "[x]"
			}
			label := fmt.Sprintf("%s %s", indicator, opt.label)
			if idx == m.cursor && opt.selectable {
				lines = append(lines, filterCursorStyle.Render(label))
			} else {
				lines = append(lines, filterInactiveStyle.Render(label))
			}
		}
	}

	help := "space toggle • ctrl+l clear • enter close"
	lines = append(lines, "", filterHelpStyle.Render(help))
	return strings.Join(lines, "\n")
}

func (m *FilterModel) moveCursor(delta int) {
	if len(m.options) == 0 || !m.hasSelectableOpts {
		return
	}

	next := m.cursor
	for {
		next += delta
		if next < 0 {
			next = len(m.options) - 1
		}
		if next >= len(m.options) {
			next = 0
		}

		if m.options[next].selectable {
			m.cursor = next
			return
		}

		if next == m.cursor {
			return
		}
	}
}

func (m *FilterModel) toggleCurrent() bool {
	if len(m.options) == 0 || m.cursor < 0 || m.cursor >= len(m.options) {
		return false
	}

	opt := m.options[m.cursor]
	if !opt.selectable {
		return false
	}

	switch {
	case opt.key == "" && opt.value != "":
		if _, ok := m.selectedTags[opt.value]; ok {
			delete(m.selectedTags, opt.value)
		} else {
			m.selectedTags[opt.value] = struct{}{}
		}
	case opt.key != "" && opt.value != "":
		if _, ok := m.selectedMetadata[opt.key]; !ok {
			m.selectedMetadata[opt.key] = make(map[string]struct{})
		}
		if _, ok := m.selectedMetadata[opt.key][opt.value]; ok {
			delete(m.selectedMetadata[opt.key], opt.value)
			if len(m.selectedMetadata[opt.key]) == 0 {
				delete(m.selectedMetadata, opt.key)
			}
		} else {
			m.selectedMetadata[opt.key][opt.value] = struct{}{}
		}
	default:
		return false
	}

	return true
}

func (m *FilterModel) selectionChangedCmd() tea.Cmd {
	snapshot := FilterSelectionChangedMsg{
		Tags:     m.SelectedTags(),
		Metadata: m.SelectedMetadata(),
	}
	return func() tea.Msg { return snapshot }
}

func (m *FilterModel) clearSelections() {
	if len(m.selectedTags) == 0 && len(m.selectedMetadata) == 0 {
		return
	}
	m.selectedTags = make(map[string]struct{})
	m.selectedMetadata = make(map[string]map[string]struct{})
}

func (m *FilterModel) SelectedTags() []string {
	out := make([]string, 0, len(m.selectedTags))
	for tag := range m.selectedTags {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func (m *FilterModel) SelectedMetadata() map[string][]string {
	out := make(map[string][]string, len(m.selectedMetadata))
	keys := make([]string, 0, len(m.selectedMetadata))
	for key := range m.selectedMetadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := make([]string, 0, len(m.selectedMetadata[key]))
		for value := range m.selectedMetadata[key] {
			values = append(values, value)
		}
		sort.Strings(values)
		out[key] = values
	}
	return out
}

func (m *FilterModel) rebuildOptions() {
	m.options = m.options[:0]
	m.hasSelectableOpts = false

	if len(m.tags) > 0 {
		m.options = append(m.options, filterOption{kind: filterOptionHeader, label: "Tags"})
		for _, tag := range m.tags {
			trimmed := strings.TrimSpace(tag)
			if trimmed == "" {
				continue
			}
			m.options = append(m.options, filterOption{
				kind:       filterOptionEntry,
				label:      fmt.Sprintf("#%s", trimmed),
				value:      trimmed,
				selectable: true,
			})
			m.hasSelectableOpts = true
		}
	} else {
		m.options = append(m.options, filterOption{kind: filterOptionHeader, label: "Tags"})
		m.options = append(m.options, filterOption{kind: filterOptionEmpty, label: "No tags indexed"})
	}

	metadataKeys := make([]string, 0, len(m.metadata))
	for key := range m.metadata {
		metadataKeys = append(metadataKeys, key)
	}
	sort.Strings(metadataKeys)

	m.options = append(m.options, filterOption{kind: filterOptionHeader, label: "Metadata"})
	if len(metadataKeys) == 0 {
		m.options = append(m.options, filterOption{kind: filterOptionEmpty, label: "No metadata indexed"})
	} else {
		for _, key := range metadataKeys {
			if ShouldExcludeMetadataKey(key) {
				continue
			}

			values := m.metadata[key]
			if len(values) == 0 {
				continue
			}
			for _, value := range values {
				trimmedValue := strings.TrimSpace(value)
				if trimmedValue == "" {
					continue
				}
				label := fmt.Sprintf("%s: %s", key, trimmedValue)
				m.options = append(m.options, filterOption{
					kind:       filterOptionEntry,
					label:      label,
					key:        key,
					value:      trimmedValue,
					selectable: true,
				})
				m.hasSelectableOpts = true
			}
		}
	}

	m.ensureCursor()
}

func (m *FilterModel) ensureCursor() {
	if len(m.options) == 0 || !m.hasSelectableOpts {
		m.cursor = 0
		return
	}

	if m.cursor >= len(m.options) {
		m.cursor = len(m.options) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	if !m.options[m.cursor].selectable {
		m.moveCursor(1)
	}
}

func (m *FilterModel) isSelected(opt filterOption) bool {
	if !opt.selectable {
		return false
	}
	if opt.key == "" {
		_, ok := m.selectedTags[opt.value]
		return ok
	}
	values, ok := m.selectedMetadata[opt.key]
	if !ok {
		return false
	}
	_, ok = values[opt.value]
	return ok
}
