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
)

func ParseNoteFiles(noteFiles []string, vaultDir string, asFileDetails bool) []list.Item {
	var items []list.Item
	for _, p := range noteFiles {
		fileWithoutVault := strings.TrimPrefix(p, vaultDir+"/")

		parts := strings.Split(fileWithoutVault, string(filepath.Separator))

		var (
			n  string
			sd string
		)

		if len(parts) < 2 {
			n = parts[0]
		} else {
			sd = parts[0]
			n = strings.Join(parts[1:], string(filepath.Separator))
		}

		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		size := info.Size()
		last := info.ModTime().Format(time.RFC1123)

		c, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		title, tags := parseFrontMatter(c, n)

		items = append(items, ListItem{
			fileName:     n,
			path:         p,
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
) (title string, tags []string) {
	// Get everything between the --- block in the markdown
	re := regexp.MustCompile(`(?ms)^---\n(.+?)\n---`)
	m := re.FindSubmatch(content)
	if len(m) < 2 {
		return "", nil
	}

	yamlContent := m[1]

	var data struct {
		Title string   `yaml:"title"`
		Tags  []string `yaml:"tags"`
	}

	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return fileName, nil
	}

	return strings.TrimSpace(data.Title), data.Tags
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

		title, _ := parseFrontMatter(content, s.fileName)
		if title != "" {
			updatedContent := bytes.Replace(content, []byte(title), []byte(newName), 1)

			if err := os.WriteFile(s.path, updatedContent, 0o644); err != nil {
				m.list.NewStatusMessage(
					statusStyle(fmt.Sprintf("Error writing file: %s", err)),
				)
				return err
			}
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

		title, _ := parseFrontMatter(content, s.fileName)
		updatedContent := content
		if title != "" {
			updatedContent = bytes.Replace(content, []byte(title), []byte(newName), 1)
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
