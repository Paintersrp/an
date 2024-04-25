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

// NewFuzzyFinder creates a new FuzzyFinder instance
func NewFuzzyFinder(vaultDir string) *FuzzyFinder {
	return &FuzzyFinder{vaultDir: vaultDir}
}

// Run executes the fuzzy finder logic
func (f *FuzzyFinder) Run() {
	f.findAndExecute("")
}

// RunWithQuery executes the fuzzy finder logic with a query
func (f *FuzzyFinder) RunWithQuery(query string) {
	f.findAndExecute(query)
}

// findAndExecute encapsulates the common logic for file finding and execution
func (f *FuzzyFinder) findAndExecute(query string) {
	files, err := f.listFiles()
	if err != nil {
		fmt.Println("Error listing files:", err)
		return
	}
	f.files = files

	idx, err := f.fuzzySelectFile(query)
	if err != nil {
		f.handleFuzzySelectError(err)
		return
	}

	f.Execute(idx)
}

// fuzzySelectFile performs fuzzy selection on files based on query
func (f *FuzzyFinder) fuzzySelectFile(query string) (int, error) {
	options := []fuzzyfinder.Option{
		fuzzyfinder.WithPreviewWindow(f.renderMarkdownPreview),
	}
	if query != "" {
		options = append(options, fuzzyfinder.WithQuery(query))
	}

	// Collect titles and tags for fuzzy selection
	var filesWithTitlesAndTags []string
	for _, file := range f.files {
		content, err := os.ReadFile(file)
		if err != nil {
			return -1, err
		}
		title, tags := parseFrontMatter(content)
		titleWithTag := fmt.Sprintf("%s [Tags: %s] ", title, strings.Join(tags, ", "))
		filesWithTitlesAndTags = append(filesWithTitlesAndTags, titleWithTag)
	}

	return fuzzyfinder.Find(f.files, func(i int) string {
		return filesWithTitlesAndTags[i]
	}, options...)
}

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

	n := &zet.ZettelkastenNote{
		VaultDir: f.vaultDir,
		SubDir:   subDir,
		Filename: strings.TrimSuffix(fileName, ".md"),
	}

	n.Open()
}

func (f *FuzzyFinder) listFiles() ([]string, error) {
	var files []string
	err := filepath.Walk(f.vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (f *FuzzyFinder) renderMarkdownPreview(i, w, h int) string {
	if i == -1 {
		return ""
	}
	content, err := os.ReadFile(f.files[i])
	if err != nil {
		return "Error reading file"
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(100),
		glamour.WithColorProfile(termenv.ANSI256),
	)
	markdown, err := r.Render(string(content))
	if err != nil {
		return "Error rendering markdown"
	}
	return markdown
}

func (f *FuzzyFinder) handleFuzzySelectError(err error) {
	if err == fuzzyfinder.ErrAbort {
		fmt.Println("No file selected")
	} else {
		fmt.Println("Error selecting file:", err)
	}
}

// parseFrontMatter extracts title and tags from YAML front matter
func parseFrontMatter(content []byte) (title string, tags []string) {
	re := regexp.MustCompile(`(?ms)^---\n(.+?)\n---`)
	match := re.FindSubmatch(content)
	if len(match) < 2 {
		return "", nil
	}
	yamlContent := match[1]

	var data struct {
		Title string   `yaml:"title"`
		Tags  []string `yaml:"tags"`
	}
	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return "", nil
	}

	return strings.TrimSpace(data.Title + ".md"), data.Tags
}
