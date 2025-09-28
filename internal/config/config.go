package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Paintersrp/an/internal/pin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type PinMap map[string]string

type SearchConfig struct {
	EnableBody             bool                `yaml:"enable_body"             json:"enable_body"`
	IgnoredFolders         []string            `yaml:"ignored_folders"         json:"ignored_folders"`
	DefaultTagFilters      []string            `yaml:"tag_filters"             json:"tag_filters"`
	DefaultMetadataFilters map[string][]string `yaml:"metadata_filters"        json:"metadata_filters"`
}

type CommandTemplate struct {
	Exec    string   `yaml:"exec"    json:"exec"`
	Args    []string `yaml:"args"    json:"args"`
	Wait    *bool    `yaml:"wait"    json:"wait"`
	Silence *bool    `yaml:"silence" json:"silence"`
}

type HookConfig struct {
	PreOpen    []CommandTemplate `yaml:"pre_open"    json:"pre_open"`
	PostOpen   []CommandTemplate `yaml:"post_open"   json:"post_open"`
	PostCreate []CommandTemplate `yaml:"post_create" json:"post_create"`
}

type CaptureMatcher struct {
	Template       string `yaml:"template"        json:"template"`
	UpstreamPrefix string `yaml:"upstream_prefix" json:"upstream_prefix"`
}

type CaptureAction struct {
	Clipboard   bool           `yaml:"clipboard"    json:"clipboard"`
	Tags        []string       `yaml:"tags"         json:"tags"`
	FrontMatter map[string]any `yaml:"front_matter" json:"front_matter"`
}

type CaptureRule struct {
	Match  CaptureMatcher `yaml:"match"  json:"match"`
	Action CaptureAction  `yaml:"action" json:"action"`
}

type CaptureConfig struct {
	Rules []CaptureRule `yaml:"rules" json:"rules"`
}

type ReviewConfig struct {
	Enable     bool   `yaml:"enable"    json:"enable"`
	Directory  string `yaml:"directory" json:"directory"`
	enabledSet bool   `yaml:"-"         json:"-"`
}

func (cfg *ReviewConfig) UnmarshalYAML(value *yaml.Node) error {
	type plain ReviewConfig
	var raw plain
	if err := value.Decode(&raw); err != nil {
		return err
	}
	*cfg = ReviewConfig(raw)
	if value.Kind == yaml.MappingNode {
		for i := 0; i < len(value.Content); i += 2 {
			if strings.EqualFold(value.Content[i].Value, "enable") {
				cfg.enabledSet = true
				break
			}
		}
	}
	return nil
}

type Workspace struct {
	PinManager     *pin.PinManager           `yaml:"-"`
	NamedPins      PinMap                    `yaml:"named_pins"       json:"named_pins"`
	NamedTaskPins  PinMap                    `yaml:"named_task_pins"  json:"named_task_pins"`
	VaultDir       string                    `yaml:"vaultdir"         json:"vault_dir"`
	Editor         string                    `yaml:"editor"           json:"editor"`
	NvimArgs       string                    `yaml:"nvimargs"         json:"nvim_args"`
	FileSystemMode string                    `yaml:"fsmode"           json:"fs_mode"`
	PinnedFile     string                    `yaml:"pinned_file"      json:"pinned_file"`
	PinnedTaskFile string                    `yaml:"pinned_task_file" json:"pinned_task_file"`
	SubDirs        []string                  `yaml:"subdirs"          json:"sub_dirs"`
	Search         SearchConfig              `yaml:"search"           json:"search"`
	Views          map[string]ViewDefinition `yaml:"views"           json:"views"`
	ViewOrder      []string                  `yaml:"view_order"      json:"view_order"`
	EditorTemplate CommandTemplate           `yaml:"editor_template" json:"editor_template"`
	Hooks          HookConfig                `yaml:"hooks"           json:"hooks"`
	Review         ReviewConfig              `yaml:"review"          json:"review"`
	Capture        CaptureConfig             `yaml:"capture"         json:"capture"`
}

type Config struct {
	Workspaces       map[string]*Workspace `yaml:"workspaces"         json:"workspaces"`
	CurrentWorkspace string                `yaml:"current_workspace" json:"current_workspace"`

	active *Workspace `yaml:"-"`
}

type ViewSort struct {
	Field string `yaml:"field" json:"field"`
	Order string `yaml:"order" json:"order"`
}

type ViewDefinition struct {
	Include    []string `yaml:"include"    json:"include"`
	Exclude    []string `yaml:"exclude"    json:"exclude"`
	Sort       ViewSort `yaml:"sort"       json:"sort"`
	Predicates []string `yaml:"predicates" json:"predicates"`
}

const (
	defaultWorkspaceName = "default"
	defaultReviewDir     = "reviews"
)

var ValidModes = map[string]bool{
	"strict":  true,
	"confirm": true,
	"free":    true,
}

var validEditorNames = []string{"nvim", "obsidian", "vscode", "code", "vim", "nano", "custom"}

var ValidEditors = func() map[string]bool {
	editors := make(map[string]bool, len(validEditorNames))
	for _, editor := range validEditorNames {
		editors[editor] = true
	}

	return editors
}()

func ValidateEditor(editor string) error {
	if _, valid := ValidEditors[editor]; valid {
		return nil
	}

	return fmt.Errorf(
		"invalid editor: %q. Please choose from %s.",
		editor,
		validEditorList(),
	)
}

func validEditorList() string {
	quoted := make([]string, len(validEditorNames))
	for i, name := range validEditorNames {
		quoted[i] = fmt.Sprintf("'%s'", name)
	}

	if len(quoted) == 0 {
		return ""
	}

	if len(quoted) == 1 {
		return quoted[0]
	}

	return strings.Join(quoted[:len(quoted)-1], ", ") + ", or " + quoted[len(quoted)-1]
}

type legacyConfig struct {
	NamedPins      PinMap                    `yaml:"named_pins"`
	NamedTaskPins  PinMap                    `yaml:"named_task_pins"`
	VaultDir       string                    `yaml:"vaultdir"`
	Editor         string                    `yaml:"editor"`
	NvimArgs       string                    `yaml:"nvimargs"`
	FileSystemMode string                    `yaml:"fsmode"`
	PinnedFile     string                    `yaml:"pinned_file"`
	PinnedTaskFile string                    `yaml:"pinned_task_file"`
	SubDirs        []string                  `yaml:"subdirs"`
	Search         SearchConfig              `yaml:"search"`
	Views          map[string]ViewDefinition `yaml:"views"`
	ViewOrder      []string                  `yaml:"view_order"`
}

func newWorkspace() *Workspace {
	ws := &Workspace{
		NamedPins:      make(PinMap),
		NamedTaskPins:  make(PinMap),
		Search:         SearchConfig{EnableBody: true, DefaultMetadataFilters: make(map[string][]string)},
		Views:          make(map[string]ViewDefinition),
		ViewOrder:      nil,
		FileSystemMode: "strict",
		Review:         ReviewConfig{Enable: true, Directory: defaultReviewDir},
		Capture:        CaptureConfig{Rules: []CaptureRule{}},
	}
	ws.PinManager = pin.NewPinManager(
		pin.PinMap(ws.NamedPins),
		pin.PinMap(ws.NamedTaskPins),
		ws.PinnedFile,
		ws.PinnedTaskFile,
	)
	return ws
}

func (ws *Workspace) ensureDefaults() {
	if ws.NamedPins == nil {
		ws.NamedPins = make(PinMap)
	}
	if ws.NamedTaskPins == nil {
		ws.NamedTaskPins = make(PinMap)
	}
	if ws.Search.DefaultMetadataFilters == nil {
		ws.Search.DefaultMetadataFilters = make(map[string][]string)
	}
	if ws.Views == nil {
		ws.Views = make(map[string]ViewDefinition)
	}
	if ws.Capture.Rules == nil {
		ws.Capture.Rules = []CaptureRule{}
	}
	if !ws.Review.enabledSet && !ws.Review.Enable {
		ws.Review.Enable = true
	}
	ws.Review.Directory = strings.TrimSpace(ws.Review.Directory)
	if ws.Review.Enable {
		if ws.Review.Directory == "" {
			if pin := strings.TrimSpace(ws.NamedPins["review"]); pin != "" {
				ws.Review.Directory = pin
			} else {
				ws.Review.Directory = defaultReviewDir
			}
		}
	}
	if ws.PinManager == nil {
		ws.PinManager = pin.NewPinManager(
			pin.PinMap(ws.NamedPins),
			pin.PinMap(ws.NamedTaskPins),
			ws.PinnedFile,
			ws.PinnedTaskFile,
		)
	}
}

func Load(home string) (*Config, error) {
	path := GetConfigPath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if len(strings.TrimSpace(string(data))) == 0 {
		cfg.Workspaces = map[string]*Workspace{
			defaultWorkspaceName: newWorkspace(),
		}
		cfg.CurrentWorkspace = defaultWorkspaceName
	} else {
		raw := make(map[string]interface{})
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, err
		}

		if _, ok := raw["workspaces"]; ok {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		} else {
			var legacy legacyConfig
			if err := yaml.Unmarshal(data, &legacy); err != nil {
				return nil, err
			}
			cfg = migrateLegacyConfig(&legacy)
		}
	}

	if err := cfg.ensureInitialized(); err != nil {
		return nil, err
	}

	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return nil, err
	}

	if ws.Editor != "" {
		if err := ValidateEditor(ws.Editor); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func migrateLegacyConfig(legacy *legacyConfig) *Config {
	ws := newWorkspace()
	ws.NamedPins = legacy.NamedPins
	ws.NamedTaskPins = legacy.NamedTaskPins
	ws.VaultDir = legacy.VaultDir
	ws.Editor = legacy.Editor
	ws.NvimArgs = legacy.NvimArgs
	ws.FileSystemMode = legacy.FileSystemMode
	if ws.FileSystemMode == "" {
		ws.FileSystemMode = "strict"
	}
	ws.PinnedFile = legacy.PinnedFile
	ws.PinnedTaskFile = legacy.PinnedTaskFile
	ws.SubDirs = legacy.SubDirs
	if legacy.Search.EnableBody || legacy.Search.IgnoredFolders != nil || legacy.Search.DefaultTagFilters != nil || legacy.Search.DefaultMetadataFilters != nil {
		ws.Search = legacy.Search
		if ws.Search.DefaultMetadataFilters == nil {
			ws.Search.DefaultMetadataFilters = make(map[string][]string)
		}
	}
	if legacy.Views != nil {
		ws.Views = legacy.Views
	}
	ws.ViewOrder = legacy.ViewOrder
	ws.ensureDefaults()

	return &Config{
		Workspaces: map[string]*Workspace{
			defaultWorkspaceName: ws,
		},
		CurrentWorkspace: defaultWorkspaceName,
		active:           ws,
	}
}

func (cfg *Config) ensureInitialized() error {
	if cfg.Workspaces == nil {
		cfg.Workspaces = make(map[string]*Workspace)
	}

	if cfg.CurrentWorkspace == "" {
		if len(cfg.Workspaces) == 0 {
			cfg.Workspaces[defaultWorkspaceName] = newWorkspace()
			cfg.CurrentWorkspace = defaultWorkspaceName
		} else {
			for name := range cfg.Workspaces {
				cfg.CurrentWorkspace = name
				break
			}
		}
	}

	return cfg.setActiveWorkspace(cfg.CurrentWorkspace)
}

func (cfg *Config) setActiveWorkspace(name string) error {
	if name == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}
	ws, ok := cfg.Workspaces[name]
	if !ok {
		return fmt.Errorf("workspace %q does not exist", name)
	}
	if ws == nil {
		ws = newWorkspace()
		cfg.Workspaces[name] = ws
	}

	ws.ensureDefaults()
	cfg.CurrentWorkspace = name
	cfg.active = ws

	cfg.syncViperWithActiveWorkspace()

	return nil
}

func (cfg *Config) syncViperWithActiveWorkspace() {
	if cfg.active == nil {
		return
	}

	syncWorkspaceWithViper(cfg.active)
}

func syncWorkspaceWithViper(ws *Workspace) {
	viper.Set("vaultdir", ws.VaultDir)
	viper.Set("vaultDir", ws.VaultDir)
	viper.Set("editor", ws.Editor)
	viper.Set("nvimargs", ws.NvimArgs)
	viper.Set("fsmode", ws.FileSystemMode)
	viper.Set("pinned_file", ws.PinnedFile)
	viper.Set("pinned_task_file", ws.PinnedTaskFile)
	viper.Set("editor_template", ws.EditorTemplate)
	viper.Set("workspace_hooks", ws.Hooks)
	viper.Set("review", ws.Review)
	viper.Set("capture", ws.Capture)
	if ws.Capture.Rules == nil {
		viper.Set("capture_rules", []CaptureRule{})
	} else {
		viper.Set("capture_rules", append([]CaptureRule(nil), ws.Capture.Rules...))
	}
	if ws.SubDirs == nil {
		viper.Set("subdirs", []string{})
	} else {
		viper.Set("subdirs", append([]string(nil), ws.SubDirs...))
	}
}

func (cfg *Config) ActiveWorkspace() (*Workspace, error) {
	if cfg.active != nil {
		return cfg.active, nil
	}

	if cfg.CurrentWorkspace == "" {
		return nil, fmt.Errorf("no workspace is currently selected")
	}

	if err := cfg.setActiveWorkspace(cfg.CurrentWorkspace); err != nil {
		return nil, err
	}

	return cfg.active, nil
}

func (cfg *Config) MustWorkspace() *Workspace {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		panic(err)
	}
	return ws
}

func (cfg *Config) WorkspaceNames() []string {
	names := make([]string, 0, len(cfg.Workspaces))
	for name := range cfg.Workspaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cfg *Config) SwitchWorkspace(name string) error {
	if err := cfg.setActiveWorkspace(name); err != nil {
		return err
	}
	return cfg.Save()
}

func (cfg *Config) ActivateWorkspace(name string) error {
	return cfg.setActiveWorkspace(name)
}

func (cfg *Config) AddWorkspace(name string, ws *Workspace, makeCurrent bool) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}

	if cfg.Workspaces == nil {
		cfg.Workspaces = make(map[string]*Workspace)
	}

	if _, exists := cfg.Workspaces[trimmed]; exists {
		return fmt.Errorf("workspace %q already exists", trimmed)
	}

	if ws == nil {
		ws = newWorkspace()
	}
	ws.ensureDefaults()
	cfg.Workspaces[trimmed] = ws

	if cfg.CurrentWorkspace == "" || makeCurrent {
		if err := cfg.setActiveWorkspace(trimmed); err != nil {
			return err
		}
	}

	return cfg.Save()
}

func (cfg *Config) RemoveWorkspace(name string) error {
	if len(cfg.Workspaces) <= 1 {
		return fmt.Errorf("cannot remove the last workspace")
	}

	if _, exists := cfg.Workspaces[name]; !exists {
		return fmt.Errorf("workspace %q does not exist", name)
	}

	delete(cfg.Workspaces, name)

	if cfg.CurrentWorkspace == name {
		cfg.active = nil
		cfg.CurrentWorkspace = ""
		if err := cfg.ensureInitialized(); err != nil {
			return err
		}
	}

	return cfg.Save()
}

func (cfg *Config) GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return GetConfigPath(homeDir)
}

func (cfg *Config) AddSubdir(name string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	for _, subDir := range ws.SubDirs {
		if subDir == name {
			return fmt.Errorf("subdirectory %q already exists", name)
		}
	}

	ws.SubDirs = append(ws.SubDirs, name)
	return cfg.Save()
}

func (cfg *Config) AddView(name string, view ViewDefinition) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("view name cannot be empty")
	}

	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	if ws.Views == nil {
		ws.Views = make(map[string]ViewDefinition)
	}

	ws.Views[name] = view
	ws.ViewOrder = appendViewOrder(ws.ViewOrder, name)

	return cfg.Save()
}

func (cfg *Config) RemoveView(name string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	if ws.Views == nil {
		return fmt.Errorf("no views are configured")
	}

	if _, ok := ws.Views[name]; !ok {
		return fmt.Errorf("view %q does not exist", name)
	}

	delete(ws.Views, name)
	ws.ViewOrder = removeFromOrder(ws.ViewOrder, name)

	return cfg.Save()
}

func (cfg *Config) SetViewOrder(order []string) error {
	deduped := make([]string, 0, len(order))
	seen := make(map[string]struct{}, len(order))

	for _, name := range order {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}

		if _, exists := seen[trimmed]; exists {
			continue
		}

		seen[trimmed] = struct{}{}
		deduped = append(deduped, trimmed)
	}

	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	ws.ViewOrder = deduped
	return cfg.Save()
}

func appendViewOrder(order []string, name string) []string {
	filtered := removeFromOrder(order, name)
	return append(filtered, name)
}

func removeFromOrder(order []string, target string) []string {
	if len(order) == 0 {
		return order
	}

	filtered := order[:0]
	for _, name := range order {
		if name == target {
			continue
		}
		filtered = append(filtered, name)
	}

	return filtered
}

func (cfg *Config) ChangeMode(mode string) error {
	if _, valid := ValidModes[mode]; !valid {
		return fmt.Errorf(
			"invalid mode: %q. Please choose from 'strict', 'confirm', or 'free'",
			mode,
		)
	}

	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	ws.FileSystemMode = mode
	return cfg.Save()
}

func (cfg *Config) ChangeEditor(editor string) error {
	if err := ValidateEditor(editor); err != nil {
		return err
	}

	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	ws.Editor = editor
	return cfg.Save()
}

func (cfg *Config) AddPin(pinName, file, pinType string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	err = ws.PinManager.AddPin(pinName, file, pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) ChangePin(file, pinType, pinName string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	err = ws.PinManager.ChangePin(file, pinType, pinName)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) DeleteNamedPin(pinName, pinType string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	err = ws.PinManager.DeleteNamedPin(pinName, pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) ClearPinnedFile(pinType string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	err = ws.PinManager.ClearPinnedFile(pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) RenamePin(oldName, newName, pinType string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	err = ws.PinManager.RenamePin(oldName, newName, pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) ListPins(pinType string) error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	err = ws.PinManager.ListPins(pinType)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Config) Save() error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	if ws.Editor != "" {
		if err := ValidateEditor(ws.Editor); err != nil {
			return err
		}
	}

	cfg.syncViperWithActiveWorkspace()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configPath := cfg.GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o644)
}

func (ws *Workspace) syncPins() {
	if ws.PinManager == nil {
		return
	}
	ws.NamedPins = PinMap(ws.PinManager.NamedPins)
	ws.NamedTaskPins = PinMap(ws.PinManager.NamedTaskPins)
	ws.PinnedFile = ws.PinManager.PinnedFile
	ws.PinnedTaskFile = ws.PinManager.PinnedTaskFile
}

func (cfg *Config) syncPins() error {
	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}
	ws.syncPins()
	return nil
}

func (cfg *Config) syncPinsAndSave() error {
	if err := cfg.syncPins(); err != nil {
		return err
	}
	return cfg.Save()
}

func (cfg *Config) HandleSubdir(subdirName string) {
	ws := cfg.MustWorkspace()
	exists := ws.HasSubdir(subdirName)

	switch ws.FileSystemMode {
	case "strict":
		if !exists {
			fmt.Println("Error: Subdirectory", subdirName, "does not exist.")
			fmt.Println(
				"In strict mode, new subdirectories are included with the add-subdir command.",
			)
			os.Exit(1)
		}
	case "free":
		if !exists {
			cobra.CheckErr(cfg.AddSubdir(subdirName))
		}
	case "confirm":
		if !exists {
			cfg.getConfirmation(subdirName)
		}
	default:
		if !exists {
			cfg.getConfirmation(subdirName)
		}
	}
}

func (cfg *Config) getConfirmation(subdirName string) {
	var response string
	for {
		fmt.Printf(
			"Subdirectory %s does not exist.\nDo you want to create it?\n(y/n): ",
			subdirName,
		)
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "yes", "y":
			cobra.CheckErr(cfg.AddSubdir(subdirName))
			return
		case "no", "n":
			fmt.Println("Exiting due to non-existing subdirectory")
			os.Exit(0)
		default:
			fmt.Println("Invalid response. Please enter 'y'/'yes' or 'n'/'no'.")
		}
	}
}

func (ws *Workspace) HasSubdir(name string) bool {
	for _, subdir := range ws.SubDirs {
		if subdir == name {
			return true
		}
	}
	return false
}
