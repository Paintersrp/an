package zet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type ZettelkastenNote struct {
	VaultDir     string
	Filename     string
	OriginalTags []string
}

func NewZettelkastenNote(vaultDir string, filename string, tags []string) *ZettelkastenNote {
	return &ZettelkastenNote{
		VaultDir:     vaultDir,
		Filename:     filename,
		OriginalTags: tags,
	}
}

func (note *ZettelkastenNote) GetFilepath() string {
	// Return the filename of the note.
	return fmt.Sprintf("%s/%s.md", note.VaultDir, note.Filename)
}

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

// TODO use templates rather than building frontmatter manually
func (note *ZettelkastenNote) Create() (bool, error) {
	// Create the note file.
	file, err := os.Create(note.GetFilepath())
	if err != nil {
		return false, err
	}

	// Set up the YAML frontmatter.
	frontmatter := fmt.Sprintf("---\ntitle: %s\ndate: %s\ntags:\n", note.Filename, time.Now().Format("2006-01-02"))

	// Iterate over tags and add them to the frontmatter.
	for _, tag := range note.OriginalTags {
		frontmatter += fmt.Sprintf("  - %s\n", tag)
	}

	// Add the rest of the frontmatter.
	frontmatter += "---\n\n\n## Links:\n\n"
	file.WriteString(frontmatter)

	return true, nil
}

func (note *ZettelkastenNote) Open() error {
	exists, filePath, existsErr := note.FileExists()

	if existsErr != nil {
		return existsErr
	}

	if !exists {
		fmt.Println("error: Note with given title does not exist in the vault directory.")
		fmt.Println("hint: Try again with a new title, or run 'zet-cli open [title]' again with a create (-c) flag to create an empty note forcefully.")
		os.Exit(1)
	}

	fmt.Println("Opening file:", filePath)

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
