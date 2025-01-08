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

func ParseFrontMatter(
	content []byte,
) (title string, tags []string) {
	// Get everything between the ---s
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
