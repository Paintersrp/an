package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/parser"
)

var titlePrefixMap = map[string]string{
	"default":     "✅ - All",
	"orphan":      "❓ - Orphan",
	"unfulfilled": "⬜ - Unfulfilled",
	"archive":     "📦 - Archive",
	"trash":       "🗑️  - Trash", // Note the extra space before the dash
}

var sortFieldDisplay = map[SortField]string{
	SortFieldTitle:        "Title",
	SortFieldSubdirectory: "Subdirectory",
	SortFieldModified:     "Modified",
}

// SortField represents the available sort fields for a view.
type SortField string

const (
	SortFieldTitle        SortField = "title"
	SortFieldSubdirectory SortField = "subdirectory"
	SortFieldModified     SortField = "modified"
)

var validSortFields = map[SortField]struct{}{
	SortFieldTitle:        {},
	SortFieldSubdirectory: {},
	SortFieldModified:     {},
}

// SortOrder represents the direction of the sort.
type SortOrder string

const (
	SortOrderAscending  SortOrder = "asc"
	SortOrderDescending SortOrder = "desc"
)

var validSortOrders = map[SortOrder]struct{}{
	SortOrderAscending:  {},
	SortOrderDescending: {},
}

// Predicate represents a content based filter that can be applied to a view.
type Predicate string

const (
	PredicateOrphan      Predicate = "orphan"
	PredicateUnfulfilled Predicate = "unfulfilled"
)

var validPredicates = map[Predicate]struct{}{
	PredicateOrphan:      {},
	PredicateUnfulfilled: {},
}

// SortDefinition captures the default sort configuration for a view.
type SortDefinition struct {
	Field SortField
	Order SortOrder
}

// View represents a configuration for a specific view.
type View struct {
	Name            string
	ExcludeDirs     []string
	ExcludeFiles    []string
	IncludePatterns []string
	ExcludePatterns []string
	Predicates      []Predicate
	Sort            SortDefinition
}

// ViewManager manages available views and their configurations.
type ViewManager struct {
	Views   map[string]View
	Handler *handler.FileHandler

	config    *config.Config
	workspace *config.Workspace
	vaultDir  string
	order     []string
}

var defaultViewOrder = []string{"default", "unfulfilled", "archive", "orphan", "trash"}

// NewViewManager creates a new ViewManager instance with default views merged with user configuration.
func NewViewManager(h *handler.FileHandler, cfg *config.Config) (*ViewManager, error) {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return nil, err
	}

	vm := &ViewManager{
		Views:     make(map[string]View),
		Handler:   h,
		config:    cfg,
		workspace: ws,
	}

	if ws != nil {
		vm.vaultDir = ws.VaultDir
	}

	if vm.vaultDir == "" {
		return nil, fmt.Errorf("vault directory is not configured")
	}

	if err := vm.reload(); err != nil {
		return nil, err
	}

	return vm, nil
}

// GetTitleForView returns a formatted title for the view and sort configuration.
func GetTitleForView(viewFlag string, sortField SortField, sortOrder SortOrder) string {
	prefix, ok := titlePrefixMap[viewFlag]
	if !ok {
		prefix = titlePrefixMap["default"]
	}

	sortFieldStr, ok := sortFieldDisplay[sortField]
	if !ok {
		sortFieldStr = "Unknown"
	}

	orderStr := "Ascending"
	if sortOrder == SortOrderDescending {
		orderStr = "Descending"
	}

	return fmt.Sprintf(
		"%s View \nSort: %s (%s)",
		prefix,
		sortFieldStr,
		orderStr,
	)
}

// GetView returns the view configuration for the provided name.
func (vm *ViewManager) GetView(viewName string) (View, error) {
	view, ok := vm.Views[viewName]
	if !ok {
		return View{}, fmt.Errorf("invalid view: %s", viewName)
	}
	return view, nil
}

// GetAvailableViews returns a comma-separated list of available view names respecting the configured order.
func (vm *ViewManager) GetAvailableViews() string {
	if len(vm.order) == 0 {
		names := make([]string, 0, len(vm.Views))
		for name := range vm.Views {
			names = append(names, name)
		}
		return strings.Join(names, ", ")
	}

	return strings.Join(vm.order, ", ")
}

// Order returns a copy of the configured view order.
func (vm *ViewManager) Order() []string {
	order := make([]string, len(vm.order))
	copy(order, vm.order)
	return order
}

// NextView returns the next view name in the configured order.
func (vm *ViewManager) NextView(current string) string {
	if len(vm.order) == 0 {
		return current
	}

	for idx, name := range vm.order {
		if name == current {
			return vm.order[(idx+1)%len(vm.order)]
		}
	}

	return vm.order[0]
}

// AddCustomView persists a custom view definition and reloads the manager.
func (vm *ViewManager) AddCustomView(name string, definition config.ViewDefinition) error {
	if vm.config == nil {
		return fmt.Errorf("configuration is not available")
	}

	if err := vm.config.AddView(name, definition); err != nil {
		return err
	}

	return vm.reload()
}

// RemoveCustomView removes a custom view definition and reloads the manager.
func (vm *ViewManager) RemoveCustomView(name string) error {
	if vm.config == nil {
		return fmt.Errorf("configuration is not available")
	}

	if err := vm.config.RemoveView(name); err != nil {
		return err
	}

	return vm.reload()
}

// GetFilesByView returns the files for the requested view applying include/exclude patterns and predicates.
func (vm *ViewManager) GetFilesByView(viewFlag string) ([]string, error) {
	view, ok := vm.Views[viewFlag]
	if !ok {
		availableViews := vm.GetAvailableViews()
		return nil, fmt.Errorf(
			"invalid view: %s. Available views are: %s",
			viewFlag,
			availableViews,
		)
	}

	excludeDirs := view.ExcludeDirs
	excludeFiles := view.ExcludeFiles

	if vm.Handler == nil {
		return nil, fmt.Errorf("file handler is not configured")
	}

	files, err := vm.Handler.WalkFiles(excludeDirs, excludeFiles, "")
	if err != nil {
		return nil, err
	}

	filtered, err := vm.applyFilters(view, files)
	if err != nil {
		return nil, err
	}

	return filtered, nil
}

// VaultDir returns the vault directory associated with the manager.
func (vm *ViewManager) VaultDir() string {
	return vm.vaultDir
}

func (vm *ViewManager) reload() error {
	builtins := vm.defaultViews()

	views := make(map[string]View, len(builtins))
	for name, view := range builtins {
		views[name] = view
	}

	if vm.workspace != nil {
		for name, definition := range vm.workspace.Views {
			view, err := vm.viewFromDefinition(name, definition)
			if err != nil {
				return fmt.Errorf("view %s: %w", name, err)
			}

			views[name] = view
		}
	}

	vm.Views = views
	vm.order = vm.computeOrder(views)

	return nil
}

func (vm *ViewManager) defaultViews() map[string]View {
	vaultDir := vm.vaultDir
	if vaultDir == "" && vm.workspace != nil {
		vaultDir = vm.workspace.VaultDir
	}

	var archiveExclude, trashExclude []string
	if vm.Handler != nil {
		archiveExclude = vm.Handler.GetSubdirectories(vaultDir, "archive")
		trashExclude = vm.Handler.GetSubdirectories(vaultDir, "trash")
	}

	views := map[string]View{
		"default": {
			Name:        "default",
			ExcludeDirs: []string{"archive", "trash"},
			Sort:        SortDefinition{Field: SortFieldModified, Order: SortOrderDescending},
		},
		"orphan": {
			Name:        "orphan",
			ExcludeDirs: []string{"archive", "trash"},
			Predicates:  []Predicate{PredicateOrphan},
			Sort:        SortDefinition{Field: SortFieldModified, Order: SortOrderDescending},
		},
		"unfulfilled": {
			Name:        "unfulfilled",
			ExcludeDirs: []string{"archive", "trash"},
			Predicates:  []Predicate{PredicateUnfulfilled},
			Sort:        SortDefinition{Field: SortFieldModified, Order: SortOrderDescending},
		},
		"archive": {
			Name:        "archive",
			ExcludeDirs: archiveExclude,
			Sort:        SortDefinition{Field: SortFieldModified, Order: SortOrderDescending},
		},
		"trash": {
			Name:        "trash",
			ExcludeDirs: trashExclude,
			Sort:        SortDefinition{Field: SortFieldModified, Order: SortOrderDescending},
		},
	}

	return views
}

func (vm *ViewManager) viewFromDefinition(name string, definition config.ViewDefinition) (View, error) {
	sortDef, err := parseSort(definition.Sort)
	if err != nil {
		return View{}, err
	}

	includePatterns := sanitizePatterns(definition.Include)
	if err := validatePatterns(includePatterns); err != nil {
		return View{}, err
	}

	excludePatterns := sanitizePatterns(definition.Exclude)
	if err := validatePatterns(excludePatterns); err != nil {
		return View{}, err
	}

	predicates, err := parsePredicates(definition.Predicates)
	if err != nil {
		return View{}, err
	}

	return View{
		Name:            name,
		IncludePatterns: includePatterns,
		ExcludePatterns: excludePatterns,
		Predicates:      predicates,
		Sort:            sortDef,
	}, nil
}

func (vm *ViewManager) computeOrder(views map[string]View) []string {
	var base []string
	if vm.workspace != nil && len(vm.workspace.ViewOrder) > 0 {
		base = vm.workspace.ViewOrder
	} else {
		base = defaultViewOrder
	}

	seen := make(map[string]struct{}, len(views))
	order := make([]string, 0, len(views))

	for _, name := range base {
		if _, ok := views[name]; !ok {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		order = append(order, name)
	}

	for name := range views {
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		order = append(order, name)
	}

	return order
}

func (vm *ViewManager) applyFilters(view View, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return paths, nil
	}

	type fileInfo struct {
		abs string
		rel string
	}

	infos := make([]fileInfo, 0, len(paths))
	for _, path := range paths {
		rel, err := filepath.Rel(vm.vaultDir, path)
		if err != nil {
			return nil, fmt.Errorf("failed to determine relative path for %s: %w", path, err)
		}

		infos = append(infos, fileInfo{abs: path, rel: filepath.ToSlash(rel)})
	}

	if len(view.IncludePatterns) > 0 {
		filtered := infos[:0]
		for _, info := range infos {
			matched, err := matchAnyPattern(info.rel, view.IncludePatterns)
			if err != nil {
				return nil, err
			}
			if matched {
				filtered = append(filtered, info)
			}
		}
		infos = filtered
	}

	if len(view.ExcludePatterns) > 0 {
		filtered := infos[:0]
		for _, info := range infos {
			matched, err := matchAnyPattern(info.rel, view.ExcludePatterns)
			if err != nil {
				return nil, err
			}
			if matched {
				continue
			}
			filtered = append(filtered, info)
		}
		infos = filtered
	}

	results := make([]string, len(infos))
	for i, info := range infos {
		results[i] = info.abs
	}

	if len(view.Predicates) == 0 {
		return results, nil
	}

	filteredPaths := make([]string, 0, len(results))
	for _, path := range results {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		if matchesPredicates(content, view.Predicates) {
			filteredPaths = append(filteredPaths, path)
		}
	}

	return filteredPaths, nil
}

func parseSort(sort config.ViewSort) (SortDefinition, error) {
	field := strings.ToLower(strings.TrimSpace(sort.Field))
	order := strings.ToLower(strings.TrimSpace(sort.Order))

	sortField := SortField(field)
	if field == "" {
		sortField = SortFieldModified
	} else if !IsValidSortField(sortField) {
		return SortDefinition{}, fmt.Errorf("invalid sort field: %s", sort.Field)
	}

	sortOrder := SortOrder(order)
	if order == "" {
		sortOrder = SortOrderDescending
	} else if !IsValidSortOrder(sortOrder) {
		return SortDefinition{}, fmt.Errorf("invalid sort order: %s", sort.Order)
	}

	return SortDefinition{Field: sortField, Order: sortOrder}, nil
}

func parsePredicates(values []string) ([]Predicate, error) {
	if len(values) == 0 {
		return nil, nil
	}

	predicates := make([]Predicate, 0, len(values))
	seen := make(map[Predicate]struct{}, len(values))

	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}

		predicate := Predicate(trimmed)
		if !IsValidPredicate(predicate) {
			return nil, fmt.Errorf("invalid predicate: %s", value)
		}

		if _, exists := seen[predicate]; exists {
			continue
		}

		seen[predicate] = struct{}{}
		predicates = append(predicates, predicate)
	}

	return predicates, nil
}

func sanitizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return nil
	}

	sanitized := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		sanitized = append(sanitized, trimmed)
	}

	if len(sanitized) == 0 {
		return nil
	}

	return sanitized
}

func validatePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if _, err := filepath.Match(pattern, ""); err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
	}
	return nil
}

func matchAnyPattern(relPath string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := matchesPattern(relPath, pattern)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}

func matchesPattern(relPath, pattern string) (bool, error) {
	normalizedPattern := filepath.ToSlash(pattern)
	normalizedPath := filepath.ToSlash(relPath)

	if strings.ContainsAny(normalizedPattern, "*?[") {
		return filepath.Match(normalizedPattern, normalizedPath)
	}

	normalizedPattern = strings.TrimSuffix(normalizedPattern, "/")
	if normalizedPattern == "" {
		return false, nil
	}

	if normalizedPath == normalizedPattern {
		return true, nil
	}

	if strings.HasPrefix(normalizedPath, normalizedPattern+"/") {
		return true, nil
	}

	return false, nil
}

func matchesPredicates(content []byte, predicates []Predicate) bool {
	for _, predicate := range predicates {
		switch predicate {
		case PredicateOrphan:
			if parser.HasNoteLinks(content) {
				return false
			}
		case PredicateUnfulfilled:
			if !parser.CheckFulfillment(content, "false") {
				return false
			}
		}
	}

	return true
}

// IsValidSortField reports whether the provided sort field is supported.
func IsValidSortField(field SortField) bool {
	_, ok := validSortFields[field]
	return ok
}

// IsValidSortOrder reports whether the provided sort order is supported.
func IsValidSortOrder(order SortOrder) bool {
	_, ok := validSortOrders[order]
	return ok
}

// IsValidPredicate reports whether the predicate is supported.
func IsValidPredicate(predicate Predicate) bool {
	_, ok := validPredicates[predicate]
	return ok
}
