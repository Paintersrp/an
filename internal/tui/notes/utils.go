package notes

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"gopkg.in/yaml.v2"

	"github.com/Paintersrp/an/internal/pathutil"
)

func ParseNoteFiles(noteFiles []string, vaultDir string, asFileDetails bool) []list.Item {
	var items []list.Item
	for _, p := range noteFiles {
		normalizedPath := pathutil.NormalizePath(p)
		subDir, remainder, err := pathutil.VaultRelativeComponents(vaultDir, normalizedPath)
		if err != nil {
			continue
		}

		if remainder == "" {
			remainder = filepath.Base(normalizedPath)
			subDir = ""
		}

		sd := filepath.FromSlash(subDir)
		n := strings.ReplaceAll(remainder, "/", string(filepath.Separator))

		info, err := os.Stat(normalizedPath)
		if err != nil {
			continue
		}
		size := info.Size()
		last := info.ModTime().Format(time.RFC1123)

		c, err := os.ReadFile(normalizedPath)
		if err != nil {
			continue
		}

		title, tags, _, _ := parseFrontMatter(c, n)

		items = append(items, ListItem{
			fileName:     n,
			path:         normalizedPath,
			size:         size,
			lastModified: last,
			title:        title,
			tags:         tags,
			showFullPath: asFileDetails,
			subdirectory: sd,
		})
	}
	return items
}

func parseFrontMatter(
	content []byte,
	fileName string,
) (title string, tags []string, start, end int) {
	// Get everything between the --- block in the markdown
	re := regexp.MustCompile(`(?ms)^---\n(.+?)\n---`)
	loc := re.FindSubmatchIndex(content)
	if len(loc) < 4 {
		return "", nil, -1, -1
	}

	start = loc[2]
	end = loc[3]
	yamlContent := content[start:end]

	var data yaml.MapSlice

	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return fileName, nil, start, end
	}

	for _, item := range data {
		key, ok := item.Key.(string)
		if !ok {
			continue
		}

		switch key {
		case "title":
			if value, ok := item.Value.(string); ok {
				title = strings.TrimSpace(value)
			}
		case "tags":
			switch value := item.Value.(type) {
			case []interface{}:
				for _, tag := range value {
					if tagStr, ok := tag.(string); ok {
						tags = append(tags, tagStr)
					}
				}
			case []string:
				tags = append(tags, value...)
			}
		}
	}

	return title, tags, start, end
}

func updateFrontMatterTitle(content []byte, start, end int, newTitle string) ([]byte, bool, error) {
	if start < 0 || end < 0 || start >= end || end > len(content) {
		return content, false, nil
	}

	original := content[start:end]

	var data yaml.MapSlice
	if err := yaml.Unmarshal(original, &data); err != nil {
		return content, false, nil
	}

	updated := false
	for i, item := range data {
		key, ok := item.Key.(string)
		if !ok {
			continue
		}
		if key == "title" {
			data[i].Value = newTitle
			updated = true
			break
		}
	}

	if !updated {
		return content, false, nil
	}

	marshaled, err := yaml.Marshal(data)
	if err != nil {
		return content, false, err
	}

	trailingNewlines := 0
	for trailingNewlines < len(original) && original[len(original)-1-trailingNewlines] == '\n' {
		trailingNewlines++
	}

	marshaled = bytes.TrimSuffix(marshaled, []byte("\n"))
	if trailingNewlines > 0 {
		marshaled = append(marshaled, bytes.Repeat([]byte("\n"), trailingNewlines)...)
	}

	var buf bytes.Buffer
	buf.Grow(len(content) - (end - start) + len(marshaled))
	buf.Write(content[:start])
	buf.Write(marshaled)
	buf.Write(content[end:])

	return buf.Bytes(), true, nil
}

func renameFile(m NoteListModel) error {
	newName := m.inputModel.Input.Value()

	if s, ok := m.list.SelectedItem().(ListItem); ok {
		newPath := filepath.Join(filepath.Dir(s.path), newName+".md")
		needsRename := newPath != s.path

		if needsRename {
			if _, err := os.Stat(newPath); err == nil {
				m.list.NewStatusMessage(
					statusStyle(fmt.Sprintf("File already exists: %s", newName+".md")),
				)
				return fmt.Errorf("destination file %q already exists: %w", newPath, fs.ErrExist)
			} else if !errors.Is(err, fs.ErrNotExist) {
				m.list.NewStatusMessage(
					statusStyle(fmt.Sprintf("Error checking destination file: %s", err)),
				)
				return err
			}
		}

		content, err := os.ReadFile(s.path)
		if err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error reading file: %s", err)),
			)
			return err
		}

		_, _, start, end := parseFrontMatter(content, s.fileName)
		if updatedContent, updated, err := updateFrontMatterTitle(content, start, end, newName); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error updating title: %s", err)),
			)
			return err
		} else if updated {
			if err := os.WriteFile(s.path, updatedContent, 0o644); err != nil {
				m.list.NewStatusMessage(
					statusStyle(fmt.Sprintf("Error writing file: %s", err)),
				)
				return err
			}
			content = updatedContent
		}

		if needsRename {
			if err := os.Rename(s.path, newPath); err != nil {
				m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Error renaming: %s", err)))
				return err
			}
		}
	}
	return nil
}

func copyFile(m NoteListModel) error {
	newName := m.inputModel.Input.Value()

	if s, ok := m.list.SelectedItem().(ListItem); ok {
		newPath := filepath.Join(filepath.Dir(s.path), newName+".md")

		if _, err := os.Stat(newPath); err == nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("File already exists: %s", newName+".md")),
			)
			return fmt.Errorf("destination file %q already exists: %w", newPath, fs.ErrExist)
		} else if !errors.Is(err, fs.ErrNotExist) {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error checking destination file: %s", err)),
			)
			return err
		}

		content, err := os.ReadFile(s.path)
		if err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error reading file: %s", err)),
			)
			return err
		}

		_, _, start, end := parseFrontMatter(content, s.fileName)
		updatedContent, _, err := updateFrontMatterTitle(content, start, end, newName)
		if err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error updating title: %s", err)),
			)
			return err
		}

		destFile, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error creating destination file: %s", err)),
			)
			return err
		}
		defer destFile.Close()

		if _, err := destFile.Write(updatedContent); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error writing to destination file: %s", err)),
			)
			return err
		}

		m.list.NewStatusMessage(statusStyle("File copied and title updated successfully"))
	}
	return nil
}

func castToListItems(items []list.Item) []ListItem {
	var listItems []ListItem
	for _, item := range items {
		if listItem, ok := item.(ListItem); ok {
			listItems = append(listItems, listItem)
		}
	}
	return listItems
}
