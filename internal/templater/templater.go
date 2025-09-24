// Package templater provides functionality to manage and render templates for Zettelkasten notes.
package templater

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Paintersrp/an/internal/config"
)

//go:embed templates
var embeddedTemplates embed.FS

// AvailableTemplates defines the set of templates that are available for use
var AvailableTemplates = map[string]bool{
	"day":             true,
	"roadmap":         true,
	"zet":             true,
	"project":         true,
	"project-release": true,
	"feature":         true,
	"stack":           true,
	"question":        true,
	"version":         true,
	"week":            true,
	"month":           true,
	"echo":            true,
	"sum":             true,
	"year":            true,
	"review-daily":    true,
	"review-weekly":   true,
	"review-retro":    true,
}

type SingleTemplate struct {
	FilePath string
	Content  string
	Manifest TemplateManifest
}

type TemplateMap map[string]SingleTemplate

// Templater manages a collection of templates.
type Templater struct {
	templates TemplateMap
	resolved  map[string]resolvedTemplate
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
	Metadata  map[string]interface{}
}

type TemplateManifest struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Preview     string          `yaml:"preview"`
	Extends     []string        `yaml:"extends"`
	Fields      []TemplateField `yaml:"fields"`
}

type TemplateField struct {
	Key      string   `yaml:"key"`
	Label    string   `yaml:"label"`
	Prompt   string   `yaml:"prompt"`
	Type     string   `yaml:"type"`
	Options  []string `yaml:"options"`
	Required bool     `yaml:"required"`
	Default  string   `yaml:"default"`
	Defaults []string `yaml:"defaults"`
	Multi    bool     `yaml:"multi"`
}

type resolvedTemplate struct {
	Content  string
	Manifest TemplateManifest
}

// Templates returns the list of template names known to the templater.
func (t *Templater) Templates() []string {
	names := make([]string, 0, len(t.templates))
	for name := range t.templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func NewTemplater(workspace *config.Workspace) (*Templater, error) {
	tmplMap := make(TemplateMap)

	if workspace != nil {
		workspaceTemplateDir := filepath.Join(workspace.VaultDir, ".an", "templates")
		if err := tmplMap.loadTemplates(workspaceTemplateDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	// Load user templates first to give them precedence.
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	userTemplateDir := filepath.Join(userHomeDir, ".an", "templates")

	if err := tmplMap.loadTemplates(userTemplateDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	for templateName := range tmplMap {
		AvailableTemplates[templateName] = true
	}

	err = tmplMap.loadEmbeddedTemplates(embeddedTemplates)
	if err != nil {
		return nil, err
	}

	return &Templater{templates: tmplMap, resolved: make(map[string]resolvedTemplate)}, nil
}

// Execute finds the template by name, validates the data against the expected struct, and renders the template.
func (t *Templater) Execute(templateName string, data interface{}) (string, error) {
	resolved, err := t.resolveTemplate(templateName)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(templateName).Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(resolved.Content)
	if err != nil {
		return "", err
	}

	var renderedTemplate bytes.Buffer
	if err := tmpl.Execute(&renderedTemplate, data); err != nil {
		return "", err
	}

	templData, ok := data.(TemplateData)
	if !ok {
		return renderedTemplate.String(), nil
	}

	updated, err := injectMetadata(renderedTemplate.String(), templData.Metadata)
	if err != nil {
		return "", err
	}

	return updated, nil
}

// GenerateTagsAndDate generates the Zettelkasten-style timestamp and auto-generated tags.
func (t *Templater) GenerateTagsAndDate(tmplName string) (string, []string) {
	cur := time.Now().UTC()
	zetTime := cur.Format("20060102150405")

	day := strings.ToLower(cur.Weekday().String())
	hour := fmt.Sprintf("%02dh", cur.Hour())

	switch tmplName {
	case "day", "daily":
		return zetTime, []string{"daily", day, hour}
	default:
		return zetTime, []string{}
	}
}

func (m TemplateMap) loadEmbeddedTemplates(embeddedFS embed.FS) error {
	return fs.WalkDir(
		embeddedFS,
		"templates",
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
				if _, exists := m[name]; !exists {
					data, err := fs.ReadFile(embeddedFS, path)
					if err != nil {
						return err
					}

					manifest, body, err := parseManifest(name, string(data))
					if err != nil {
						return err
					}

					m[name] = SingleTemplate{
						FilePath: path,
						Manifest: manifest,
						Content:  body,
					}
				}
			}

			return nil
		},
	)
}

func (m TemplateMap) loadTemplates(dirPath string) error {

	return filepath.Walk(
		dirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && filepath.Ext(path) == ".tmpl" {
				name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

				if _, exists := m[name]; !exists {
					contents, readErr := os.ReadFile(path)
					if readErr != nil {
						return readErr
					}

					manifest, body, err := parseManifest(name, string(contents))
					if err != nil {
						return err
					}

					m[name] = SingleTemplate{
						FilePath: path,
						Manifest: manifest,
						Content:  body,
					}
				}
			}
			return nil
		},
	)
}

func parseManifest(name, content string) (TemplateManifest, string, error) {
	manifest := TemplateManifest{Name: name}
	cleaned := content

	manifestPattern := regexp.MustCompile(`(?s)^\s*\{\{/\*\s*an:manifest\s*(.*?)\*/\}\}`)
	if loc := manifestPattern.FindStringSubmatchIndex(content); loc != nil {
		raw := content[loc[2]:loc[3]]
		if err := yaml.Unmarshal([]byte(raw), &manifest); err != nil {
			return manifest, "", fmt.Errorf("failed to parse manifest for template %s: %w", name, err)
		}
		if manifest.Name == "" {
			manifest.Name = name
		}
		cleaned = content[loc[1]:]
	}

	manifest.Extends = append([]string{}, manifest.Extends...)
	manifest.Fields = normalizeFields(manifest.Fields)

	return manifest, strings.TrimLeft(cleaned, "\r\n"), nil
}

func normalizeFields(fields []TemplateField) []TemplateField {
	normalized := make([]TemplateField, 0, len(fields))
	for _, field := range fields {
		if field.Key == "" {
			continue
		}
		if field.Prompt == "" {
			if field.Label != "" {
				field.Prompt = field.Label
			} else {
				field.Prompt = strings.Title(strings.ReplaceAll(field.Key, "_", " "))
			}
		}
		if field.Label == "" {
			field.Label = field.Prompt
		}
		if field.Type == "" {
			field.Type = "text"
		}
		normalized = append(normalized, field)
	}
	return normalized
}

func (t *Templater) resolveTemplate(name string) (resolvedTemplate, error) {
	return t.resolveTemplateWithVisited(name, make(map[string]bool))
}

func (t *Templater) resolveTemplateWithVisited(name string, visited map[string]bool) (resolvedTemplate, error) {
	if cached, ok := t.resolved[name]; ok {
		return cached, nil
	}

	if visited[name] {
		return resolvedTemplate{}, fmt.Errorf("detected circular template inheritance involving %s", name)
	}
	visited[name] = true
	defer delete(visited, name)

	tmplData, ok := t.templates[name]
	if !ok {
		return resolvedTemplate{}, errors.New("template not found")
	}

	combined := tmplData.Manifest
	combined.Fields = nil

	var builder strings.Builder
	var collectedFields []TemplateField
	previews := make([]string, 0)

	for _, parent := range tmplData.Manifest.Extends {
		parentResolved, err := t.resolveTemplateWithVisited(parent, visited)
		if err != nil {
			return resolvedTemplate{}, err
		}
		builder.WriteString(parentResolved.Content)
		collectedFields = append(collectedFields, parentResolved.Manifest.Fields...)
		if parentResolved.Manifest.Preview != "" {
			previews = append(previews, parentResolved.Manifest.Preview)
		}
	}

	builder.WriteString(tmplData.Content)

	collectedFields = mergeFields(collectedFields, tmplData.Manifest.Fields)
	combined.Fields = collectedFields

	if combined.Preview == "" && len(previews) > 0 {
		combined.Preview = strings.Join(previews, "\n---\n")
	}

	resolved := resolvedTemplate{
		Content:  builder.String(),
		Manifest: combined,
	}

	if t.resolved == nil {
		t.resolved = make(map[string]resolvedTemplate)
	}
	t.resolved[name] = resolved

	return resolved, nil
}

func mergeFields(existing, overrides []TemplateField) []TemplateField {
	merged := make([]TemplateField, 0, len(existing)+len(overrides))
	index := make(map[string]int)

	for _, field := range existing {
		if field.Key == "" {
			continue
		}
		index[field.Key] = len(merged)
		merged = append(merged, field)
	}

	for _, field := range overrides {
		if field.Key == "" {
			continue
		}
		if pos, ok := index[field.Key]; ok {
			merged[pos] = field
			continue
		}
		index[field.Key] = len(merged)
		merged = append(merged, field)
	}

	return merged
}

func injectMetadata(rendered string, metadata map[string]interface{}) (string, error) {
	if len(metadata) == 0 {
		return rendered, nil
	}

	if strings.HasPrefix(rendered, "---\n") {
		parts := strings.SplitN(strings.TrimPrefix(rendered, "---\n"), "\n---\n", 2)
		if len(parts) == 2 {
			fm := parts[0]
			body := parts[1]
			updated, err := mergeFrontMatter(fm, metadata)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("---\n%s---\n%s", updated, body), nil
		}
	}

	serialized, err := marshalFrontMatter(metadata)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n%s", serialized, rendered), nil
}

func mergeFrontMatter(frontMatter string, metadata map[string]interface{}) (string, error) {
	existing := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(frontMatter), &existing); err != nil {
		return "", err
	}

	for key, value := range metadata {
		existing[key] = value
	}

	return marshalFrontMatter(existing)
}

func marshalFrontMatter(data map[string]interface{}) (string, error) {
	ordered := make(map[string]interface{}, len(data))
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		ordered[key] = data[key]
	}

	buf, err := yaml.Marshal(ordered)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// Manifest returns the resolved manifest for the provided template, applying inheritance rules.
func (t *Templater) Manifest(name string) (TemplateManifest, error) {
	resolved, err := t.resolveTemplate(name)
	if err != nil {
		return TemplateManifest{}, err
	}
	return resolved.Manifest, nil
}

// Preview returns either the manifest-defined preview or a truncated rendering of the template content.
func (t *Templater) Preview(name string) (string, error) {
	resolved, err := t.resolveTemplate(name)
	if err != nil {
		return "", err
	}

	if resolved.Manifest.Preview != "" {
		return resolved.Manifest.Preview, nil
	}

	lines := strings.Split(resolved.Content, "\n")
	const maxLines = 20
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	return strings.Join(lines, "\n"), nil
}
