// Package zet provides functionality for managing zettelkasten (atomic) notes.
package zet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Paintersrp/an/internal/templater"
	"github.com/spf13/viper"
)

// TODO:ensure "zet" / "atoms" directory in vault

// ZettelkastenNote represents a zettelkasten note with its metadata.
type ZettelkastenNote struct {
	VaultDir     string
	SubDir       string
	Filename     string
	OriginalTags []string
}

func NewZettelkastenNote(
	vaultDir string,
	subDir string,
	filename string,
	tags []string,
) *ZettelkastenNote {
	return &ZettelkastenNote{
		VaultDir:     vaultDir,
		SubDir:       subDir,
		Filename:     filename,
		OriginalTags: tags,
	}
}

// GetFilepath returns the file path of the zettelkasten note.
func (note *ZettelkastenNote) GetFilepath() string {
	return fmt.Sprintf(
		"%s/%s/%s.md",
		note.VaultDir,
		note.SubDir,
		note.Filename,
	)
}

// EnsurePath creates the necessary directory structure for the note file.
func (note *ZettelkastenNote) EnsurePath() (string, error) {
	dir := fmt.Sprintf("%s/%s", note.VaultDir, note.SubDir)
	filePath := fmt.Sprintf("%s/%s.md", dir, note.Filename)

	// Check if the directory already exists
	_, err := os.Stat(dir)
	if err == nil {
		// Directory already exists, return file path
		return filePath, nil
	}

	// If the directory does not exist, create it
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			// Failed to create directory
			return "", err
		}
		return filePath, nil
	}

	// Other error occurred
	return "", err
}

// Because of the way the fuzzy finding works, we should never get a file exists error
// as we are only selecting from the files processed directly from the vault
// If you open into a file exists error, there's an issue with the options being
// provided by the fuzzyfinder
//
// FileExists checks if the Zettelkasten note file already exists.
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
func (note *ZettelkastenNote) Create(tmplName string, t *templater.Templater) (bool, error) {
	// Verify the directories up to the new note
	path, pathErr := note.EnsurePath()
	if pathErr != nil {
		return false, pathErr // exit
	}

	// Create the empty note file.
	file, createErr := os.Create(path)
	if createErr != nil {
		return false, createErr // exit
	}

	// Setup template metadata
	zetTime, tags := t.GenerateTagsAndDate(tmplName)
	data := templater.TemplateData{
		Title: note.Filename,
		Date:  zetTime,
		Tags:  append(note.OriginalTags, tags...),
	}

	// Execute the template and return the rendered output
	output, renderErr := t.Execute(tmplName, data)
	if renderErr != nil {
		// TODO: delete file made on failure?
		fmt.Printf("Failed to execute template: %v", renderErr)
		return false, renderErr // exit
	}

	// Write to file
	file.WriteString(output)

	// Return created (true) and nil (error)
	return true, nil
}

// Open opens the Zettelkasten note in the configured editor.
func (note *ZettelkastenNote) Open() error {
	exists, filePath, existsErr := note.FileExists()

	if existsErr != nil {
		return existsErr
	}

	// TODO: fix flag notes, as we are using molecule mode now
	if !exists {
		fmt.Println("error: Note with given title does not exist in the vault directory.")
		fmt.Println("hint: Try again with a new title, or run 'zet-cli open [title]' again with a create (-c) flag to create an empty note forcefully.")
		os.Exit(1)
	}

	fmt.Println("Opening file:", filePath)

	// TODO: eventually support more editors and therefore we need to rename nvimargs. sorry one true god
	editor := viper.GetString("editor")
	editorArgs := viper.GetString("nvimargs")

	// We will split the command into arguments
	var cmdArgs []string

	if editorArgs != "" {
		// User specified command
		cmdArgs = strings.Fields(editorArgs)
		cmdArgs = append([]string{editor, filePath}, cmdArgs...)
	} else {
		// Default to just opening nvim if no command is specified
		cmdArgs = []string{editor, filePath}
	}

	// Open the note in Editor.
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command and wait for it to finish
	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting Neovim:", err)
		return err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for Neovim to close:", err)
		return err
	}

	return nil
}
