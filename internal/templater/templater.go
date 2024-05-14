// Package templater provides functionality to manage and render templates for Zettelkasten notes.
package templater

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// TODO: template aliases?

// AvailableTemplates defines the set of templates that are available for use
var AvailableTemplates = map[string]bool{
	"day":      true,
	"roadmap":  true,
	"zet":      true,
	"project":  true,
	"feature":  true,
	"stack":    true,
	"question": true,
	"version":  true,
	"week":     true,
	"month":    true,
	"echo":     true,
	"sum":      true,
	"year":     true,
}

type SingleTemplate struct {
	FilePath string
	Data     TemplateData
}

type TemplateMap map[string]SingleTemplate

// Templater manages a collection of templates.
type Templater struct {
	templates TemplateMap
}

// TemplateData defines the structure for data that will be passed to templates during rendering.
type TemplateData struct {
	Title     string
	Date      string
	Upstream  string
	Content   string
	Tags      []string
	Links     []string
	Fulfilled bool
}

func NewTemplater() (*Templater, error) {
	tmplMap := make(TemplateMap)

	// Load user templates first to give them precedence.
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	userTemplateDir := filepath.Join(userHomeDir, ".an", "templates")
	err = tmplMap.loadTemplates(userTemplateDir)
	if err != nil {
		return nil, err
	}

	for templateName := range tmplMap {
		AvailableTemplates[templateName] = true
	}

	var templateDir string
	if os.Getenv("DEV_MODE") == "true" {
		templateDir = "./internal/templater/templates"
	} else {
		executableDir, err := os.Executable()
		if err != nil {
			return nil, err
		}
		templateDir = filepath.Join(filepath.Dir(executableDir), "internal/templater/templates")
	}

	err = tmplMap.loadTemplates(templateDir)
	if err != nil {
		return nil, err
	}

	return &Templater{templates: tmplMap}, nil
}

// Execute finds the template by name, validates the data against the expected struct, and renders the template.
func (t *Templater) Execute(templateName string, data interface{}) (string, error) {
	tmplData, ok := t.templates[templateName]
	if !ok {
		return "", errors.New("template not found")
	}

	tmpl, err := template.ParseFiles(tmplData.FilePath)
	if err != nil {
		return "", err
	}

	var renderedTemplate bytes.Buffer
	err = tmpl.Execute(&renderedTemplate, data)
	if err != nil {
		return "", err
	}

	return renderedTemplate.String(), nil
}

// GenerateTagsAndDate generates the Zettelkasten-style timestamp and auto-generated tags.
func (t *Templater) GenerateTagsAndDate(tmplName string) (string, []string) {
	cur := time.Now().UTC()
	zetTime := cur.Format("20060102150405")

	day := strings.ToLower(cur.Weekday().String())
	hour := fmt.Sprintf("%02dh", cur.Hour())

	switch tmplName {
	case "daily":
		return zetTime, []string{"daily", day, hour}
	default:
		return zetTime, []string{}
	}
}

func (m TemplateMap) loadTemplates(dirPath string) error {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(
		dirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && filepath.Ext(path) == ".tmpl" {
				name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
				if _, exists := m[name]; !exists {
					var data TemplateData
					m[name] = SingleTemplate{
						FilePath: path,
						Data:     data,
					}
				}
			}

			return nil
		},
	)
}
