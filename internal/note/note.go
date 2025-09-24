// Package note provides functionality for managing zettelkasten (atomic) notes.
package note

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/pathutil"
	"github.com/Paintersrp/an/internal/templater"
)

// ZettelkastenNote represents a zettelkasten note with its metadata.
type ZettelkastenNote struct {
	VaultDir      string
	SubDir        string
	Filename      string
	Upstream      string
	OriginalTags  []string
	OriginalLinks []string
}

// NewZettelkastenNote creates a new ZettelkastenNote instance.
func NewZettelkastenNote(
	vaultDir string,
	subDir string,
	filename string,
	tags []string,
	links []string,
	upstream string,
) *ZettelkastenNote {
	return &ZettelkastenNote{
		VaultDir:      vaultDir,
		SubDir:        subDir,
		Filename:      filename,
		OriginalTags:  tags,
		OriginalLinks: links,
		Upstream:      upstream,
	}
}

// GetFilepath returns the file path of the zettelkasten note.
func (note *ZettelkastenNote) GetFilepath() string {
	return filepath.Join(note.VaultDir, note.SubDir, note.Filename+".md")
}

// EnsurePath creates the necessary directory structure for the note file.
func (note *ZettelkastenNote) EnsurePath() (string, error) {
	dir := filepath.Join(note.VaultDir, note.SubDir)
	filePath := filepath.Join(dir, note.Filename+".md")

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return "", err
		}
	}

	return filePath, nil
}

// FileExists checks if the zettelkasten note file already exists.
func (note *ZettelkastenNote) FileExists() (bool, string, error) {
	noteFilePath := note.GetFilepath()
	_, err := os.Stat(noteFilePath)

	if err == nil {
		return true, noteFilePath, nil
	}

	if os.IsNotExist(err) {
		return false, noteFilePath, nil
	}

	return false, noteFilePath, err
}

// Create generates a new Zettelkasten note using a template.
func (note *ZettelkastenNote) Create(
	tmplName string,
	t *templater.Templater,
	content string,
	metadata map[string]interface{},
) (bool, error) {
	path, err := note.EnsurePath()
	if err != nil {
		return false, err
	}

	file, err := os.Create(path)
	if err != nil {
		return false, err
	}
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	cleanup := func() {
		if file != nil {
			file.Close()
			file = nil
		}
		removeCreatedArtifacts(path, note.VaultDir)
	}

	zetTime, tags := t.GenerateTagsAndDate(tmplName)
	data := templater.TemplateData{
		Title:     note.Filename,
		Date:      zetTime,
		Tags:      append(note.OriginalTags, tags...),
		Links:     note.OriginalLinks,
		Upstream:  note.Upstream,
		Content:   content,
		Fulfilled: false,
		Metadata:  metadata,
	}

	output, err := t.Execute(tmplName, data)
	if err != nil {
		cleanup()
		return false, fmt.Errorf("failed to execute template: %w", err)
	}

	_, err = file.WriteString(output)
	if err != nil {
		cleanup()
		return false, fmt.Errorf("failed to write to file: %w", err)
	}

	if file != nil {
		file.Close()
		file = nil
	}

	if err := RunPostCreateHooks(path); err != nil {
		return false, fmt.Errorf("post-create hook failed: %w", err)
	}

	return true, nil
}

func removeCreatedArtifacts(filePath, vaultDir string) {
	if filePath == "" {
		return
	}

	_ = os.Remove(filePath)

	vault := filepath.Clean(vaultDir)
	dir := filepath.Dir(filePath)

	for {
		if dir == vault {
			break
		}

		rel, err := filepath.Rel(vault, dir)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			break
		}

		if err := os.Remove(dir); err != nil {
			break
		}

		dir = filepath.Dir(dir)
	}
}

// Open opens the Zettelkasten note in the configured editor.
func (note *ZettelkastenNote) Open() error {
	exists, filePath, err := note.FileExists()
	if err != nil {
		return err
	}

	// TODO: fix flag notes, as we are using molecule mode now
	if !exists {
		fmt.Println(
			"error: Note with given title does not exist in the vault directory.",
		)
		fmt.Println(
			"hint: Try again with a new title, or run 'zet-cli open [title]' again with a create (-c) flag to create an empty note forcefully.",
		)
		os.Exit(1)
	}

	if err := OpenFromPath(filePath, false); err != nil {
		// TODO: fix - print is too specific
		fmt.Println(
			"Error opening note in Neovim:",
			err,
		)
		os.Exit(1)
	}

	return nil
}

// HandleConflicts checks for file naming conflicts and provides suggestions.
func (note *ZettelkastenNote) HandleConflicts() error {
	exists, _, err := note.FileExists()
	if err != nil {
		return fmt.Errorf("error processing note file: %w", err)
	}

	if exists {
		return errors.New("note with given title already exists in the vault directory")
	}

	return nil
}

// GetNotesInDirectory retrieves all note filenames in the specified vault and subdirectory.
func GetNotesInDirectory(vaultDir, subDir string) ([]string, error) {
	dirPath := filepath.Join(vaultDir, subDir)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var notes []string
	for _, file := range files {
		if !file.IsDir() {
			notes = append(notes, strings.TrimSuffix(file.Name(), ".md"))
		}
	}

	return notes, nil
}

// TODO: return errors
// StaticHandleNoteLaunch handles the creation and opening of a note.
func StaticHandleNoteLaunch(
	note *ZettelkastenNote,
	t *templater.Templater,
	tmpl, content string,
	metadata map[string]interface{},
) {
	if metadata == nil {
		var err error
		metadata, err = CollectTemplateMetadata(t, tmpl)
		if err != nil {
			fmt.Printf("error collecting template metadata: %v\n", err)
			os.Exit(1)
		}
	}

	created, err := note.Create(tmpl, t, content, metadata)
	if err != nil {
		fmt.Printf("error creating note file: %v\n", err)
		os.Exit(1)
	}

	if created {
		if err := note.Open(); err != nil {
			fmt.Printf("error opening note in editor: %v\n", err)
			os.Exit(1)
		}
	}
}

func CollectTemplateMetadata(t *templater.Templater, templateName string) (map[string]interface{}, error) {
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	if interactive {
		execName := filepath.Base(os.Args[0])
		if strings.HasSuffix(execName, ".test") {
			interactive = false
		}
	}

	return collectTemplateMetadata(t, templateName, interactive, func() *bufio.Reader {
		if interactive {
			return bufio.NewReader(os.Stdin)
		}
		return nil
	}())
}

// CollectTemplateMetadataNonInteractive gathers metadata defaults and validates required
// fields without prompting the user for input. This is intended for callers that cannot
// interact with stdin, such as the TUI, but still need manifest processing.
func CollectTemplateMetadataNonInteractive(t *templater.Templater, templateName string) (map[string]interface{}, error) {
	return collectTemplateMetadata(t, templateName, false, nil)
}

func collectTemplateMetadata(
	t *templater.Templater,
	templateName string,
	interactive bool,
	reader *bufio.Reader,
) (map[string]interface{}, error) {
	manifest, err := t.Manifest(templateName)
	if err != nil {
		return nil, err
	}
	if len(manifest.Fields) == 0 {
		return map[string]interface{}{}, nil
	}

	answers := make(map[string]interface{})

	if interactive {
		fmt.Printf("Template %s requires additional details:\n", manifest.Name)
	}

	for _, field := range manifest.Fields {
		if field.Key == "" {
			continue
		}

		var value interface{}

		if field.Multi || len(field.Defaults) > 0 {
			defaults := field.Defaults
			if len(defaults) == 0 && field.Default != "" {
				defaults = []string{field.Default}
			}
			if !interactive {
				if len(defaults) == 0 && field.Required {
					return nil, fmt.Errorf(
						"field %q is required but interactive input is not available", field.Key,
					)
				}
				value = defaults
			} else {
				for {
					prompt := fieldPrompt(field)
					fmt.Print(prompt)
					input, _ := reader.ReadString('\n')
					input = strings.TrimSpace(input)
					if input == "" && len(defaults) > 0 {
						value = defaults
						break
					}
					if input == "" && field.Required {
						fmt.Println("This field is required. Please enter a value.")
						continue
					}
					if input == "" {
						value = []string{}
						break
					}
					entries := splitListInput(input)
					if len(field.Options) > 0 {
						if err := validateOptions(entries, field.Options); err != nil {
							fmt.Printf("%v\n", err)
							continue
						}
					}
					value = entries
					break
				}
			}
		} else {
			if !interactive {
				if field.Default != "" {
					value = field.Default
				} else if field.Required {
					return nil, fmt.Errorf(
						"field %q is required but interactive input is not available", field.Key,
					)
				} else {
					value = ""
				}
			} else {
				for {
					prompt := fieldPrompt(field)
					fmt.Print(prompt)
					input, _ := reader.ReadString('\n')
					input = strings.TrimSpace(input)
					if input == "" && field.Default != "" {
						value = field.Default
						break
					}
					if input == "" && field.Required {
						fmt.Println("This field is required. Please enter a value.")
						continue
					}
					if input == "" {
						value = ""
						break
					}
					if len(field.Options) > 0 {
						if err := validateOption(input, field.Options); err != nil {
							fmt.Printf("%v\n", err)
							continue
						}
					}
					value = input
					break
				}
			}
		}

		answers[field.Key] = value
	}

	return answers, nil
}

func fieldPrompt(field templater.TemplateField) string {
	var parts []string
	parts = append(parts, field.Prompt)
	if len(field.Options) > 0 {
		parts = append(parts, fmt.Sprintf("[%s]", strings.Join(field.Options, ", ")))
	}
	if field.Multi {
		parts = append(parts, "(comma separated)")
	}
	if len(field.Defaults) > 0 {
		parts = append(parts, fmt.Sprintf("default: %s", strings.Join(field.Defaults, ", ")))
	} else if field.Default != "" {
		parts = append(parts, fmt.Sprintf("default: %s", field.Default))
	}
	return strings.Join(parts, " ") + ": "
}

func validateOptions(values []string, options []string) error {
	optSet := make(map[string]struct{}, len(options))
	for _, opt := range options {
		optSet[strings.TrimSpace(opt)] = struct{}{}
	}
	for _, value := range values {
		if _, ok := optSet[value]; !ok {
			return fmt.Errorf("value %q is not one of the allowed options", value)
		}
	}
	return nil
}

func validateOption(value string, options []string) error {
	for _, option := range options {
		if value == option {
			return nil
		}
	}
	return fmt.Errorf("value %q is not one of the allowed options", value)
}

func splitListInput(input string) []string {
	if input == "" {
		return []string{}
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// OpenFromPath opens the note in the configured editor.
// EditorLaunch represents the command necessary to start an editor along with
// whether the caller should wait for the process to finish before resuming the
// UI.
type EditorLaunch struct {
	Cmd  *exec.Cmd
	Wait bool
}

type editorCommand struct {
	command string
	args    []string
	wait    bool
	silence bool
}

func (cmd editorCommand) launch() (*EditorLaunch, error) {
	return newEditorLaunch(cmd.command, cmd.args, cmd.wait, cmd.silence)
}

type editorTemplateContext struct {
	File     string
	Vault    string
	Relative string
	Filename string
	Editor   string
	BaseCmd  string
	BaseArgs []string
}

// EditorLaunchForPath prepares an editor command for the provided path without
// starting it. Callers can decide whether to run the command synchronously or
// asynchronously based on the returned Wait flag.
func EditorLaunchForPath(path string, obsidian bool) (*EditorLaunch, error) {
	editor := strings.TrimSpace(viper.GetString("editor"))
	if obsidian {
		editor = "obsidian"
	}

	baseCmd, baseErr := buildEditorCommand(path, editor)

	if !obsidian {
		var template config.CommandTemplate
		if err := viper.UnmarshalKey("editor_template", &template); err == nil {
			if execName := strings.TrimSpace(template.Exec); execName != "" {
				ctx := buildEditorTemplateContext(path, editor, baseCmd)
				wrapped, err := applyEditorTemplate(template, ctx, baseCmd)
				if err != nil {
					return nil, err
				}
				return wrapped.launch()
			}
		}
	}

	if baseErr != nil {
		return nil, baseErr
	}
	if baseCmd == nil {
		return nil, fmt.Errorf("unable to determine editor command")
	}

	return baseCmd.launch()
}

func buildEditorCommand(path string, editor string) (*editorCommand, error) {
	switch editor {
	case "nvim":
		return buildNvimCommand(path)
	case "vim":
		return &editorCommand{command: "vim", args: []string{path}, wait: true}, nil
	case "nano":
		return &editorCommand{command: "nano", args: []string{path}, wait: true}, nil
	case "vscode", "code":
		return buildVSCodeCommand(path)
	case "obsidian":
		return buildObsidianCommand(path)
	case "custom":
		return nil, fmt.Errorf("custom editor requires an editor_template command")
	case "":
		return nil, fmt.Errorf("editor not configured")
	default:
		return nil, fmt.Errorf("unsupported editor: %s", editor)
	}
}

func buildNvimCommand(path string) (*editorCommand, error) {
	args := []string{"nvim"}
	if extra := strings.TrimSpace(viper.GetString("nvimargs")); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}
	args = append(args, path)
	return &editorCommand{command: args[0], args: args[1:], wait: true}, nil
}

func buildVSCodeCommand(path string) (*editorCommand, error) {
	switch runtime.GOOS {
	case "darwin":
		return &editorCommand{command: "open", args: []string{"-n", "-b", "com.microsoft.VSCode", "--args", path}, wait: false, silence: true}, nil
	case "linux":
		return &editorCommand{command: "code", args: []string{path}, wait: false, silence: true}, nil
	case "windows":
		return &editorCommand{command: "cmd", args: []string{"/c", "code", path}, wait: false, silence: true}, nil
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func buildObsidianCommand(path string) (*editorCommand, error) {
	fullVaultDir := viper.GetString("vaultdir")
	normalizedVaultDir := pathutil.NormalizePath(fullVaultDir)
	vaultName := filepath.Base(normalizedVaultDir)

	relativePath, err := pathutil.VaultRelative(fullVaultDir, path)
	if err != nil {
		return nil, fmt.Errorf("unable to determine relative path for obsidian: %w", err)
	}

	obsidianURI := fmt.Sprintf("obsidian://open?vault=%s&file=%s", vaultName, relativePath)

	switch runtime.GOOS {
	case "darwin":
		return &editorCommand{command: "open", args: []string{obsidianURI}, wait: false, silence: true}, nil
	case "linux":
		return &editorCommand{command: "xdg-open", args: []string{obsidianURI}, wait: false, silence: true}, nil
	case "windows":
		return &editorCommand{command: "cmd", args: []string{"/c", "start", obsidianURI}, wait: false, silence: true}, nil
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func buildEditorTemplateContext(path string, editor string, base *editorCommand) editorTemplateContext {
	vault := viper.GetString("vaultdir")
	relative, err := pathutil.VaultRelative(vault, path)
	if err != nil {
		relative = path
	}

	ctx := editorTemplateContext{
		File:     path,
		Vault:    vault,
		Relative: relative,
		Filename: filepath.Base(path),
		Editor:   editor,
		BaseCmd:  editor,
	}

	if base != nil {
		if base.command != "" {
			ctx.BaseCmd = base.command
		}
		if len(base.args) > 0 {
			ctx.BaseArgs = append(ctx.BaseArgs, base.args...)
		}
	}

	return ctx
}

func applyEditorTemplate(template config.CommandTemplate, ctx editorTemplateContext, base *editorCommand) (*editorCommand, error) {
	execName := strings.TrimSpace(expandEditorPlaceholders(template.Exec, ctx))
	if execName == "" {
		return nil, fmt.Errorf("editor_template.exec must not be empty")
	}

	args := expandTemplateArgs(template.Args, ctx, base)

	wait := true
	silence := false
	if base != nil {
		wait = base.wait
		silence = base.silence
	}
	if template.Wait != nil {
		wait = *template.Wait
	}
	if template.Silence != nil {
		silence = *template.Silence
	}

	return &editorCommand{command: execName, args: args, wait: wait, silence: silence}, nil
}

func expandTemplateArgs(raw []string, ctx editorTemplateContext, base *editorCommand) []string {
	if len(raw) == 0 {
		return nil
	}

	var joined string
	if base != nil {
		joined = strings.Join(base.args, " ")
	}

	args := make([]string, 0, len(raw))
	for _, token := range raw {
		trimmed := strings.TrimSpace(token)
		if trimmed == "{args}" {
			if base != nil && len(base.args) > 0 {
				args = append(args, base.args...)
			}
			continue
		}

		expanded := expandEditorPlaceholders(token, ctx)
		if strings.Contains(expanded, "{args}") {
			expanded = strings.ReplaceAll(expanded, "{args}", joined)
		}
		args = append(args, expanded)
	}

	return args
}

func expandEditorPlaceholders(value string, ctx editorTemplateContext) string {
	replacements := map[string]string{
		"{file}":     ctx.File,
		"{vault}":    ctx.Vault,
		"{relative}": ctx.Relative,
		"{filename}": ctx.Filename,
		"{cmd}":      ctx.BaseCmd,
		"{editor}":   ctx.Editor,
	}

	result := value
	for placeholder, replacement := range replacements {
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	return result
}

// OpenFromPath opens the note in the configured editor.
func OpenFromPath(path string, obsidian bool) error {
	launch, err := EditorLaunchForPath(path, obsidian)
	if err != nil {
		return err
	}

	if err := RunPreOpenHooks(path); err != nil {
		return fmt.Errorf("pre-open hook failed: %w", err)
	}

	if launch.Wait {
		if launch.Cmd.Stdin == nil {
			launch.Cmd.Stdin = os.Stdin
		}
		if launch.Cmd.Stdout == nil {
			launch.Cmd.Stdout = os.Stdout
		}
		if launch.Cmd.Stderr == nil {
			launch.Cmd.Stderr = os.Stderr
		}
	}

	if err := launch.Cmd.Start(); err != nil {
		fmt.Printf("Error starting editor: %v\n", err)
		return err
	}

	if !launch.Wait {
		if err := RunPostOpenHooks(path); err != nil {
			return fmt.Errorf("post-open hook failed: %w", err)
		}
		return nil
	}

	if err := launch.Cmd.Wait(); err != nil {
		fmt.Printf("Error waiting for editor to close: %v\n", err)
		return err
	}

	if err := RunPostOpenHooks(path); err != nil {
		return fmt.Errorf("post-open hook failed: %w", err)
	}

	return nil
}

func newEditorLaunch(command string, args []string, wait bool, silence bool) (*EditorLaunch, error) {
	cmd := exec.Command(command, args...)
	if silence {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	return &EditorLaunch{Cmd: cmd, Wait: wait}, nil
}
