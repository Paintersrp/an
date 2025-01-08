// Package note provides functionality for managing zettelkasten (atomic) notes.
package note

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/constants"
	"github.com/Paintersrp/an/internal/templater"
)

type editorFinishedMsg struct{ err error }

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
	defer file.Close()

	zetTime, tags := t.GenerateTagsAndDate(tmplName)
	data := templater.TemplateData{
		Title:    note.Filename,
		Date:     zetTime,
		Tags:     append(note.OriginalTags, tags...),
		Links:    note.OriginalLinks,
		Upstream: note.Upstream,
		Content:  content,
	}

	output, err := t.Execute(tmplName, data)
	if err != nil {
		// TODO: Delete newly created note on failure
		return false, fmt.Errorf("failed to execute template: %w", err)
	}

	_, err = file.WriteString(output)
	if err != nil {
		return false, fmt.Errorf("failed to write to file: %w", err)
	}

	return true, nil
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

	if err := OpenFromPath(filePath); err != nil {
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
func OpenFromPath(path string) error {
	editor := viper.GetString("editor")
	if editor == "" {
		return fmt.Errorf("no editor configured")
	}

	if !constants.ValidEditors[editor] {
		return fmt.Errorf("unsupported editor: %s", editor)
	}

	cmdArgs := []string{editor, path}
	if args := viper.GetString("args"); args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting editor: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error waiting for editor to close: %w", err)
	}

	return nil
}

// BubbleteaOpenFromPath opens the note in the configured editor.
func BubbleteaOpenFromPath(path string) tea.Cmd {
	editor := viper.GetString("editor")

	if !constants.ValidEditors[editor] {
		// TODO: handle errors
		return nil
	}

	cmdArgs := []string{editor, path}
	if args := viper.GetString("args"); args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
	}

	return tea.ExecProcess(
		exec.Command(cmdArgs[0], cmdArgs[1:]...),
		func(err error) tea.Msg {
			return editorFinishedMsg{err}
		},
	)
}
