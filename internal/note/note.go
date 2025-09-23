// Package note provides functionality for managing zettelkasten (atomic) notes.
package note

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"

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
) {
	created, err := note.Create(tmpl, t, content)
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

// OpenFromPath opens the note in the configured editor.
func OpenFromPath(path string, obsidian bool) error {
	var editor string
	if obsidian {
		editor = "obsidian"
	} else {
		editor = viper.GetString("editor")
	}

	switch editor {
	case "nvim":
		return openWithNvim(path)
	case "vim":
		return openWithVim(path)
	case "nano":
		return openWithNano(path)
	case "vscode", "code":
		return openWithVSCode(path)
	case "obsidian":
		return openWithObsidian(path)
	default:
		return fmt.Errorf("unsupported editor: %s", editor)
	}
}

// openWithNvim opens the note in Neovim.
func openWithNvim(path string) error {
	nvimArgs := viper.GetString("nvimargs")
	cmdArgs := []string{"nvim", path}

	if nvimArgs != "" {
		cmdArgs = append(cmdArgs, strings.Fields(nvimArgs)...)
	}

	return runEditorCommand(cmdArgs)
}

// openWithObsidian opens the note in Obsidian.
func openWithObsidian(path string) error {
	fullVaultDir := viper.GetString("vaultdir")
	normalizedVaultDir := pathutil.NormalizePath(fullVaultDir)
	vaultName := filepath.Base(normalizedVaultDir)

	relativePath, err := pathutil.VaultRelative(fullVaultDir, path)
	if err != nil {
		return fmt.Errorf("unable to determine relative path for obsidian: %w", err)
	}

	obsidianURI := fmt.Sprintf(
		"obsidian://open?vault=%s&file=%s",
		vaultName,
		relativePath,
	)

	var cmdArgs []string
	switch runtime.GOOS {
	case "darwin":
		cmdArgs = []string{"open", obsidianURI}
	case "linux":
		cmdArgs = []string{"xdg-open", obsidianURI}
	case "windows":
		cmdArgs = []string{"cmd", "/c", "start", obsidianURI}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return runEditorCommand(cmdArgs)
}

// openWithVim opens the note in vim.
func openWithVim(path string) error {
	cmdArgs := []string{"vim", path}
	return runEditorCommand(cmdArgs)
}

// openWithNano opens the note in nano.
func openWithNano(path string) error {
	cmdArgs := []string{"nano", path}
	return runEditorCommand(cmdArgs)
}

// openWithVSCode opens the note in Visual Studio Code.
func openWithVSCode(path string) error {
	var cmdArgs []string
	switch runtime.GOOS {
	case "darwin":
		cmdArgs = []string{"open", "-n", "-b", "com.microsoft.VSCode", "--args", path}
	case "linux":
		cmdArgs = []string{"code", path}
	case "windows":
		cmdArgs = []string{"cmd", "/c", "code", path}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return runEditorCommand(cmdArgs)
}

// runEditorCommand runs the editor command with the provided arguments.
// runEditorCommand runs the editor command with the provided arguments.
func runEditorCommand(cmdArgs []string) error {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// If the editor is Obsidian or VSCode, we want to silence the output
	if cmdArgs[0] == "open" || cmdArgs[0] == "xdg-open" || cmdArgs[0] == "cmd" ||
		cmdArgs[0] == "code" {
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			fmt.Printf("Error opening null device: %v\n", err)
			return err
		}
		defer devNull.Close()

		cmd.Stdout = devNull
		cmd.Stderr = devNull
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error starting editor: %v\n", err)
		return err
	}

	// If the editor is VSCode or Obsidian, we do not wait for the process to finish
	if cmdArgs[0] == "open" || cmdArgs[0] == "xdg-open" || cmdArgs[0] == "cmd" ||
		cmdArgs[0] == "code" {
		return nil
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Error waiting for editor to close: %v\n", err)
		return err
	}

	return nil
}
