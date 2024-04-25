// Package templater provides functionality to manage and render templates for Zettelkasten notes.
package templater

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"time"
)

// AvailableTemplates defines the set of templates that are available for use
var AvailableTemplates = map[string]bool{
	"daily":   true,
	"roadmap": true,
	"zet":     true,
}

// SingleTemplate represents a single template file and its associated data.
type SingleTemplate struct {
	FilePath string       `json:"file_path" yaml:"file_path"` // Path to the template file.
	Data     TemplateData `json:"data"      yaml:"data"`      // Data structure to be used with the template.
}

// TemplateMap is a map of template names to SingleTemplate instances.
type TemplateMap map[string]SingleTemplate

// Templater manages a collection of templates.
type Templater struct {
	templates TemplateMap // Map of template names to their corresponding SingleTemplate.
}

// TemplateData defines the structure for data that will be passed to templates during rendering.
type TemplateData struct {
	Title string   `json:"title" yaml:"title"` // Title of the note.
	Date  string   `json:"date"  yaml:"date"`  // Date associated with the note.
	Tags  []string `json:"tags"  yaml:"tags"`  // Tags to be associated with the note.
}

// NewTemplater initializes a new Templater instance by loading template files from a specified directory.
func NewTemplater() (*Templater, error) {
	tmplMap := make(TemplateMap)

	// do we need to do this, or just target file JIT...?
	err := filepath.Walk(
		"./internal/templater/views",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err // exit
			}

			// if not a directory and extension is .tmpl, we add it to the template map
			if !info.IsDir() && filepath.Ext(path) == ".tmpl" {
				name := strings.TrimSuffix(
					info.Name(),
					filepath.Ext(info.Name()),
				)
				var data TemplateData
				tmplMap[name] = SingleTemplate{
					FilePath: path,
					Data:     data,
				}
			}
			return nil // walk on or finish
		},
	)
	if err != nil {
		return nil, err // exit
	}

	// Return templater loaded ready to execute the available templates
	return &Templater{templates: tmplMap}, nil
}

// Execute finds the template by name, validates the data against the expected struct, and renders the template.
func (t *Templater) Execute(
	templateName string,
	data interface{},
) (string, error) {
	tmplData, ok := t.templates[templateName]
	if !ok {
		return "", errors.New("template not found")
	}

	// Validate data against the struct type.
	// Since we are auto generating the metadata, and most of the user input is already validated...
	// Do we need this?
	expectedType := reflect.TypeOf(tmplData.Data)
	if expectedType != nil &&
		!reflect.TypeOf(data).AssignableTo(expectedType) {
		return "", errors.New(
			"provided data type does not match expected template data type",
		)
	}

	// Parse and execute the template.
	tmpl, err := template.ParseFiles(tmplData.FilePath)
	if err != nil {
		return "", err
	}

	// Execute the template and write the output into the buffer.
	var renderedTemplate bytes.Buffer
	err = tmpl.Execute(&renderedTemplate, data)
	if err != nil {
		return "", err
	}

	// Return the captured template as a string.
	return renderedTemplate.String(), nil
}

// GenerateTagsAndDate generates the Zettelkasten-style timestamp and auto-generated tags.
func (t *Templater) GenerateTagsAndDate(
	tmplName string,
) (string, []string) {
	// Get the current time in UTC.
	currentTime := time.Now().UTC()

	// Format the time as a Zettelkasten-style timestamp with 12-digit seconds resolution.
	zettelkastenTime := currentTime.Format("20060102150405")

	// Generate tags for the day of the week and the hour of the day.
	dayOfWeekTag := strings.ToLower(currentTime.Weekday().String())
	hourOfDayTag := fmt.Sprintf("%02dh", currentTime.Hour())

	// return values with template specific tags as well
	// maybe better way to handle this somehow?
	switch tmplName {
	case "daily":
		return zettelkastenTime, []string{
			"daily",
			dayOfWeekTag,
			hourOfDayTag,
		}
	default:
		return zettelkastenTime, []string{dayOfWeekTag, hourOfDayTag}
	}
}
