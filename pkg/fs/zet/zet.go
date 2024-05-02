// Package zet provides functionality for managing zettelkasten (atomic) notes.
package zet

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/spf13/viper"
)

// TODO:ensure "zet" / "atoms" directory in vault

// ZettelkastenNote represents a zettelkasten note with its metadata.
type ZettelkastenNote struct {
	VaultDir      string   `json:"vault_dir"      yaml:"vault_dir"`
	SubDir        string   `json:"sub_dir"        yaml:"sub_dir"`
	Filename      string   `json:"filename"       yaml:"filename"`
	OriginalTags  []string `json:"original_tags"  yaml:"original_tags"`
	OriginalLinks []string `json:"original_links" yaml:"original_links"`
	Upstream      string   `json:"upstream"       yaml:"upstream"`
}

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
func (note *ZettelkastenNote) Create(
	tmplName string,
	t *templater.Templater,
) (bool, error) {
	// Verify the directories up to the new note
	path, err := note.EnsurePath()
	if err != nil {
		return false, err // exit
	}

	// Create the empty note file.
	file, err := os.Create(path)
	if err != nil {
		return false, err // exit
	}

	// Setup template metadata
	zetTime, tags := t.GenerateTagsAndDate(tmplName)
	data := templater.TemplateData{
		Title:    note.Filename,
		Date:     zetTime,
		Tags:     append(note.OriginalTags, tags...),
		Links:    note.OriginalLinks,
		Upstream: note.Upstream,
	}

	// Execute the template and return the rendered output
	output, err := t.Execute(tmplName, data)
	if err != nil {
		// TODO: delete file made on failure?
		fmt.Printf("Failed to execute template: %v", err)
		return false, err // exit
	}

	// Write to file
	file.WriteString(output)

	// Return created (true) and nil (error)
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
		fmt.Println(
			"Error opening note in Neovim:",
			err,
		)
		os.Exit(1)
	}

	return nil
}

func (note *ZettelkastenNote) HandleConflicts() error {
	exists, _, err := note.FileExists()
	if err != nil {
		fmt.Printf("error processing note file: %s", err)
		return err
	}

	if exists {
		fmt.Println("error: Note with given title already exists in the vault directory.")
		fmt.Println(
			"hint: Try again with a new title, or run again with either an overwrite (-o) flag or an increment (-i) flag",
		)
		return errors.New("file naming conflict")
	}

	return nil
}

// GetNotesInDirectory retrieves all note filenames in the specified vault and subdirectory.
func GetNotesInDirectory(vaultDir, subDir string) ([]string, error) {
	var notes []string
	// Construct the directory path
	dirPath := fmt.Sprintf("%s/%s", vaultDir, subDir)
	// Read the directory contents
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	// Filter and collect note filenames
	for _, file := range files {
		if !file.IsDir() {
			notes = append(notes, file.Name())
		}
	}
	return notes, nil
}

func StaticHandleNoteLaunch(
	note *ZettelkastenNote,
	t *templater.Templater,
	tmpl string,
) {
	_, err := note.Create(tmpl, t)
	if err != nil {
		fmt.Printf("error creating note file: %s", err)
		os.Exit(1)
	}

	// Open the note in Neovim.
	if err := note.Open(); err != nil {
		fmt.Println(
			"Error opening note in Neovim:",
			err,
		)
		os.Exit(1)
	}
}

// OpenFromPath opens the note in the configured editor.
func OpenFromPath(path string) error {
	editor := viper.GetString("editor")

	switch editor {
	case "nvim":
		return OpenWithNvim(path)
	case "obsidian":
		return OpenWithObsidian(path)
	default:
		fmt.Printf("Error: Unsupported editor '%s'\n", editor)
		return fmt.Errorf("unsupported editor: %s", editor)
	}
}

// OpenWithNvim opens the note in Neovim.
func OpenWithNvim(path string) error {
	nvimArgs := viper.GetString("nvimargs")
	cmdArgs := []string{"nvim", path}

	if nvimArgs != "" {
		// Append user-specified arguments for Neovim
		cmdArgs = append(cmdArgs, strings.Fields(nvimArgs)...)
	}

	return runEditorCommand(cmdArgs)
}

// OpenWithObsidian opens the note in Obsidian.
func OpenWithObsidian(path string) error {
	// Get the full vault directory path from the configuration
	fullVaultDir := viper.GetString("vaultdir")
	fmt.Printf("config vault dir: %s", fullVaultDir)

	// Extract the vault name from the full vault directory path
	vaultName := filepath.Base(fullVaultDir)
	fmt.Printf("VAULTNAME BASE: %s", vaultName)

	// Get the relative path by removing the full vault directory path from the file path
	relativePath := strings.TrimPrefix(path, fmt.Sprintf("%s/", fullVaultDir))
	fmt.Printf("relative file path: %s", relativePath)

	// Construct the obsidian URI
	obsidianURI := fmt.Sprintf(
		"obsidian://open?vault=%s&file=%s",
		vaultName,
		relativePath,
	)

	// Obsidian is opened via a URL scheme, so we use 'open' command on macOS,
	// 'xdg-open' on Linux, and 'start' on Windows.
	var cmdArgs []string
	switch runtime.GOOS {
	case "darwin":
		cmdArgs = []string{"open", obsidianURI}
	case "linux":
		cmdArgs = []string{"xdg-open", obsidianURI}
	case "windows":
		cmdArgs = []string{"cmd", "/c", "start", obsidianURI}
	default:
		fmt.Printf("Error: Unsupported operating system '%s'\n", runtime.GOOS)
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return runEditorCommand(cmdArgs)
}

// runEditorCommand runs the editor command with the provided arguments.
func runEditorCommand(cmdArgs []string) error {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// If the editor is Obsidian, we want to silence the output
	if cmdArgs[0] == "open" || cmdArgs[0] == "xdg-open" || cmdArgs[0] == "cmd" {
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

	// If the editor is Obsidian, we do not wait for the process to finish
	if cmdArgs[0] == "open" || cmdArgs[0] == "xdg-open" || cmdArgs[0] == "cmd" {
		return nil
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Error waiting for editor to close: %v\n", err)
		return err
	}

	return nil
}
