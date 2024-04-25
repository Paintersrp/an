package fzf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Paintersrp/an/internal/zet"
	"github.com/charmbracelet/glamour"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/muesli/termenv"
	"gopkg.in/yaml.v2"
)

// FuzzyFinder encapsulates the fuzzy finder functionality
type FuzzyFinder struct {
	vaultDir string
	files    []string
}

func NewFuzzyFinder(vaultDir string) *FuzzyFinder {
	return &FuzzyFinder{vaultDir: vaultDir}
}

func (f *FuzzyFinder) Run() {
	f.findAndExecute("")
}

func (f *FuzzyFinder) RunWithQuery(query string) {
	f.findAndExecute(query)
}

// findAndExecute encapsulates the common logic for file finding and execution
func (f *FuzzyFinder) findAndExecute(query string) {
	// Load the files from the targer directory for searching
	files, err := f.listFiles()
	if err != nil {
		fmt.Println("Error listing files:", err)
		return // exit
	}

	f.files = files

	idx, err := f.fuzzySelectFile(query)
	if err != nil {
		f.handleFuzzySelectError(err)
		return // exit
	}

	// Execute open into editor on given file index
	f.Execute(idx)
}

// listFiles walks the user's vault directory recursively gathering files for searching
func (f *FuzzyFinder) listFiles() ([]string, error) {
	var files []string
	err := filepath.Walk(
		f.vaultDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err // exit
			}
			// append file if not a directory
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil // walk on or finish
		},
	)

	// Return files and any errors
	return files, err
}

// fuzzySelectFile performs fuzzy selection on files based on query
func (f *FuzzyFinder) fuzzySelectFile(
	query string,
) (int, error) {
	// Initial options for the fuzzy finder, in this case our preview window with
	// our glamour formatted and styled content
	options := []fuzzyfinder.Option{
		fuzzyfinder.WithPreviewWindow(
			f.renderMarkdownPreview,
		),
	}

	// Append the query, if exists
	if query != "" {
		options = append(
			options,
			fuzzyfinder.WithQuery(query),
		)
	}

	// Collect titles and tags for fuzzy selection
	var filesWithTitlesAndTags []string
	for _, file := range f.files {
		content, err := os.ReadFile(file)
		if err != nil {
			return -1, err // no file, unlikely
		}

		// Read in markdown frontmatter
		title, tags := parseFrontMatter(content)

		// Format title for fuzzy finder display to include tags
		titleWithTag := fmt.Sprintf(
			"%s [Tags: %s] ",
			title,
			strings.Join(tags, ", "),
		)

		// Append to our array of files
		filesWithTitlesAndTags = append(
			filesWithTitlesAndTags,
			titleWithTag,
		)
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

// parseFrontMatter extracts title and tags from YAML front matter
func parseFrontMatter(
	content []byte,
) (title string, tags []string) {
	// Get everything between the ---s
	re := regexp.MustCompile(`(?ms)^---\n(.+?)\n---`)
	match := re.FindSubmatch(content)
	if len(match) < 2 {
		return "", nil // no yaml content found
	}

	yamlContent := match[1]

	// Setup struct for binding the unmarshaled yamlContent
	var data struct {
		Title string   `yaml:"title"`
		Tags  []string `yaml:"tags"`
	}

	// Bind yamlContent to data struct, or give err
	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return "", nil // no data
	}

	// Return file name and tags
	return strings.TrimSpace(data.Title + ".md"), data.Tags
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
	fileWithoutVault := strings.TrimPrefix(
		selectedFile,
		f.vaultDir+"/",
	)

	// Split the file path by the path separator
	pathParts := strings.Split(
		fileWithoutVault,
		string(filepath.Separator),
	)

	// The first part is the subdirectory
	subDir := pathParts[0]

	// The remaining parts joined together form the filename
	fileName := strings.Join(
		pathParts[1:],
		string(filepath.Separator),
	)

	// Setup temporary struct to launch with the internal Open functionality
	n := &zet.ZettelkastenNote{
		VaultDir: f.vaultDir,
		SubDir:   subDir,
		Filename: strings.TrimSuffix(fileName, ".md"),
	}

	// Opens the note in the configured editor
	n.Open()
}
