// Package note provides functionality for managing zettelkasten (atomic) notes.
package note

import (
	"errors"
	"fmt"
	"io"
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
// EditorLaunch represents the command necessary to start an editor along with
// whether the caller should wait for the process to finish before resuming the
// UI.
type EditorLaunch struct {
	Cmd  *exec.Cmd
	Wait bool
}

// EditorLaunchForPath prepares an editor command for the provided path without
// starting it. Callers can decide whether to run the command synchronously or
// asynchronously based on the returned Wait flag.
func EditorLaunchForPath(path string, obsidian bool) (*EditorLaunch, error) {
	var editor string
	if obsidian {
		editor = "obsidian"
	} else {
		editor = viper.GetString("editor")
	}

	switch editor {
	case "nvim":
		return launchWithNvim(path)
	case "vim":
		return newEditorLaunch("vim", []string{path}, true, false)
	case "nano":
		return newEditorLaunch("nano", []string{path}, true, false)
	case "vscode", "code":
		return launchWithVSCode(path)
	case "obsidian":
		return launchWithObsidian(path)
	default:
		return nil, fmt.Errorf("unsupported editor: %s", editor)
	}
}

// OpenFromPath opens the note in the configured editor.
func OpenFromPath(path string, obsidian bool) error {
	launch, err := EditorLaunchForPath(path, obsidian)
	if err != nil {
		return err
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
		return nil
	}

	if err := launch.Cmd.Wait(); err != nil {
		fmt.Printf("Error waiting for editor to close: %v\n", err)
		return err
	}

	return nil
}

func launchWithNvim(path string) (*EditorLaunch, error) {
	args := []string{"nvim"}
	if extra := viper.GetString("nvimargs"); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}
	args = append(args, path)

	return newEditorLaunch(args[0], args[1:], true, false)
}

func launchWithVSCode(path string) (*EditorLaunch, error) {
	switch runtime.GOOS {
	case "darwin":
		return newEditorLaunch("open", []string{"-n", "-b", "com.microsoft.VSCode", "--args", path}, false, true)
	case "linux":
		return newEditorLaunch("code", []string{path}, false, true)
	case "windows":
		return newEditorLaunch("cmd", []string{"/c", "code", path}, false, true)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func launchWithObsidian(path string) (*EditorLaunch, error) {
	fullVaultDir := viper.GetString("vaultdir")
	normalizedVaultDir := pathutil.NormalizePath(fullVaultDir)
	vaultName := filepath.Base(normalizedVaultDir)

	relativePath, err := pathutil.VaultRelative(fullVaultDir, path)
	if err != nil {
		return nil, fmt.Errorf("unable to determine relative path for obsidian: %w", err)
	}

	if !launch.Wait {
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		return newEditorLaunch("open", []string{obsidianURI}, false, true)
	case "linux":
		return newEditorLaunch("xdg-open", []string{obsidianURI}, false, true)
	case "windows":
		return newEditorLaunch("cmd", []string{"/c", "start", obsidianURI}, false, true)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func newEditorLaunch(command string, args []string, wait bool, silence bool) (*EditorLaunch, error) {
	cmd := exec.Command(command, args...)

	if wait {
		if cmd.Stdin == nil {
			cmd.Stdin = os.Stdin
		}

		if cmd.Stdout == nil {
			if silence {
				cmd.Stdout = io.Discard
			} else {
				cmd.Stdout = os.Stdout
			}
		}

		if cmd.Stderr == nil {
			if silence {
				cmd.Stderr = io.Discard
			} else {
				cmd.Stderr = os.Stderr
			}
		}
	} else if silence {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	return &EditorLaunch{Cmd: cmd, Wait: wait}, nil
}
