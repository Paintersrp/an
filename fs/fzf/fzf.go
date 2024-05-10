package fzf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/muesli/termenv"

	"github.com/Paintersrp/an/fs/handler"
	"github.com/Paintersrp/an/fs/parser"
	"github.com/Paintersrp/an/fs/zet"
)

// FuzzyFinder encapsulates the fuzzy finder functionality
type FuzzyFinder struct {
	handler  *handler.FileHandler
	vaultDir string
	Header   string
	files    []string
}

func NewFuzzyFinder(vaultDir, header string) *FuzzyFinder {
	h := handler.NewFileHandler(vaultDir)
	return &FuzzyFinder{vaultDir: vaultDir, Header: header, handler: h}
}

func (f *FuzzyFinder) Run(execute bool) (string, error) {
	if execute {
		f.findAndExecute("")
		return "", nil
	} else {
		return f.findAndReturn("")
	}
}

func (f *FuzzyFinder) RunWithQuery(query string, execute bool) (string, error) {
	if execute {
		f.findAndExecute(query)
		return "", nil
	} else {
		return f.findAndReturn(query)
	}
}

func (f *FuzzyFinder) find(query string) (int, error) {
	files, err := f.handler.WalkFiles(nil, nil, "default")

	if err != nil {
		return -1, fmt.Errorf("error listing files: %w", err)
	}

	f.files = files

	return f.fuzzySelectFile(query)
}

// findAndReturn handles the logic of finding and returning the selected file
func (f *FuzzyFinder) findAndReturn(query string) (string, error) {
	idx, err := f.find(query)
	if err != nil {
		f.handleFuzzySelectError(err)
		return "", err
	}

	if idx == -1 {
		return "", fmt.Errorf("no file selected")
	}

	return f.files[idx], nil
}

// findAndExecute encapsulates the common logic for file finding and execution
func (f *FuzzyFinder) findAndExecute(query string) {
	idx, err := f.find(query)
	if err != nil {
		f.handleFuzzySelectError(err)
		return
	}

	if idx != -1 {
		f.Execute(idx)
	}
}

// fuzzySelectFile performs fuzzy selection on files based on query
func (f *FuzzyFinder) fuzzySelectFile(query string) (int, error) {
	options := []fuzzyfinder.Option{
		fuzzyfinder.WithPreviewWindow(f.renderMarkdownPreview),
	}

	// Append the query, if exists
	if query != "" {
		options = append(options, fuzzyfinder.WithQuery(query))
	}

	// Append the header, if exists
	if f.Header != "" {
		options = append(options, fuzzyfinder.WithHeader(f.Header))
	}

	// Collect titles and tags for fuzzy selection
	var filesWithTitlesAndTags []string
	for _, file := range f.files {
		content, err := os.ReadFile(file)
		if err != nil {
			return -1, err // no file, unlikely
		}

		// Read in markdown frontmatter
		title, tags := parser.ParseFrontMatter(content)

		if title == "" {
			title = filepath.Base(file)
		}

		titleWithTags := ""

		if len(tags) == 0 {
			titleWithTags = fmt.Sprintf(
				"%s [No tags] ",
				title,
			)
		} else {
			titleWithTags = fmt.Sprintf(
				"%s [Tags: %s] ",
				title,
				strings.Join(tags, ", "),
			)

		}

		// Append to our array of files
		filesWithTitlesAndTags = append(filesWithTitlesAndTags, titleWithTags)
	}

	// Run the find on the files, showing the formatted titles
	return fuzzyfinder.Find(f.files, func(i int) string {
		return filesWithTitlesAndTags[i]
	}, options...)
}

// renderMarkdownPreview handles rendering the colorized preview display with glamour,
// adding formatting and styling to the terminal display.
func (f *FuzzyFinder) renderMarkdownPreview(
	i, w, h int,
) string {
	if i == -1 {
		return "" // show nothing
	}

	// Read file from system
	content, err := os.ReadFile(f.files[i])
	if err != nil {
		return "Error reading file"
	}

	// Initiate glamour renderer to add colors to our markdown preview
	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(100),
		glamour.WithColorProfile(termenv.ANSI256),
	)

	// Render formatted and styled markdown content
	markdown, err := r.Render(string(content))
	if err != nil {
		return "Error rendering markdown" // Displayed in Preview Pane
	}

	// Return markdown output
	return markdown
}

// handleFuzzySelectError prints appropriate messages for fuzzy select errors
func (f *FuzzyFinder) handleFuzzySelectError(err error) {
	if err == fuzzyfinder.ErrAbort {
		fmt.Println("No file selected")
	} else {
		fmt.Println("Error selecting file:", err)
	}
}

// Execute opens the target file selected by the fuzzy finder in the configured editor with arguments
func (f *FuzzyFinder) Execute(idx int) {
	selectedFile := f.files[idx]

	// Remove the vault directory from the file path
	fileWithoutVault := strings.TrimPrefix(selectedFile, f.vaultDir+"/")

	// Split the file path by the path separator
	pathParts := strings.Split(fileWithoutVault, string(filepath.Separator))

	// The first part is the subdirectory
	subDir := pathParts[0]

	// The remaining parts joined together form the filename
	fileName := strings.Join(pathParts[1:], string(filepath.Separator))

	// Setup temporary struct to launch with the internal Open functionality
	n := &zet.ZettelkastenNote{
		VaultDir: f.vaultDir,
		SubDir:   subDir,
		Filename: strings.TrimSuffix(fileName, ".md"),
	}

	// Opens the note in the configured editor
	n.Open()
}
