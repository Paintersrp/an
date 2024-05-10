package parser

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v1"
)

func HasNoteLinks(content []byte) bool {
	re := regexp.MustCompile(`\[\[.+\]\]`)
	return re.Match(content)
}

// check = "true" for fulfilled
// check = "false" for unfulfilled
func CheckFulfillment(content []byte, check string) bool {
	re := regexp.MustCompile(`(?m)^fulfilled:\s*(true|false)$`)
	matches := re.FindSubmatch(content)
	if len(matches) > 1 {
		return string(matches[1]) == check
	}
	return false
}

// parseFrontMatter extracts title and tags from YAML front matter
func ParseFrontMatter(
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
