package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/yaml.v1"
)

// TODO: Tags are being added in but not removed correctly
// TODO: Linked Notes still need handling
// TODO: General Cleanup

type Frontmatter struct {
	Title       string   `yaml:"title"`
	Upstream    string   `yaml:"up"`
	CreatedAt   string   `yaml:"created"`
	Tags        []string `yaml:"tags"`
	LinkedNotes []string
}

func GetMarkdownFiles(tree *object.Tree) ([]string, error) {
	var files []string

	treeWalker := object.NewTreeWalker(tree, true, nil)
	defer treeWalker.Close()

	for {
		name, entry, err := treeWalker.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !entry.Mode.IsFile() {
			continue
		}

		if filepath.Ext(name) == ".md" {
			files = append(files, name)
		}
	}

	return files, nil
}

func UpdateVaultCommit(
	commitName string,
	s *state.State,
) error {

	data := map[string]interface{}{
		"name":   s.Config.ActiveVault,
		"commit": commitName,
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return err
	}

	req, err := http.NewRequest(
		"PATCH",
		fmt.Sprintf("http://localhost:6474/v1/api/vaults/%d", s.Config.VaultID),
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		fmt.Printf("failed to create request: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Println("status code: ", resp.StatusCode)
		return nil
	}

	var respData interface{}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("failed to decode response: %v", err)
		return err
	}

	return nil
}

func SyncNotes(
	vaultPath string,
	repo *git.Repository,
	commit plumbing.Hash,
	s *state.State,
	claims *utils.Claims,
) error {
	currentCommitObj, err := repo.CommitObject(commit)
	if err != nil {
		return err
	}

	parents := currentCommitObj.Parents()
	defer parents.Close()
	parentCommitObj, err := parents.Next()
	if err != nil {
		return err
	}

	patch, err := parentCommitObj.Patch(currentCommitObj)
	if err != nil {
		return err
	}

	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()
		var oldFilePath, newFilePath string
		fileDeleted := false

		if to != nil && from != nil {
			// File was modified
			oldFilePath = from.Path()
			newFilePath = to.Path()
		} else if to != nil {
			// File was added
			newFilePath = to.Path()
		} else if from != nil {
			// File was deleted
			oldFilePath = from.Path()
			err := SyncNoteDelete(oldFilePath, s, claims)
			if err != nil {
				fmt.Printf("failed to delete note %s: %v", oldFilePath, err)
				continue
			}
			fileDeleted = true
		}

		if !fileDeleted &&
			(filepath.Ext(oldFilePath) == ".md" || filepath.Ext(newFilePath) == ".md") {
			err := SyncNoteUpdate(vaultPath, oldFilePath, newFilePath, s, claims)
			if err != nil {
				fmt.Printf("failed to sync note %s: %v", newFilePath, err)
				continue
			}
		}
	}

	return nil
}

func SyncNotesInit(
	vaultPath string,
	repo *git.Repository,
	commit plumbing.Hash,
	s *state.State,
	claims *utils.Claims,
) error {
	commitObj, err := repo.CommitObject(commit)
	if err != nil {
		return err
	}

	tree, err := commitObj.Tree()
	if err != nil {
		return err
	}

	files, err := GetMarkdownFiles(tree)
	if err != nil {
		return err
	}

	for _, file := range files {
		err := SyncNote(vaultPath, file, s, claims)
		if err != nil {
			fmt.Printf("failed to sync note %s: %v", file, err)
			continue
		}
	}

	return nil
}

func SyncNoteDelete(noteTitle string, s *state.State, claims *utils.Claims) error {
	data := map[string]interface{}{
		"title":   strings.TrimSuffix(filepath.Base(noteTitle), ".md"),
		"user_id": claims.UserID,
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return err
	}

	fmt.Println(string(dataJson))

	req, err := http.NewRequest(
		"DELETE",
		"http://localhost:6474/v1/api/notes",
		bytes.NewBuffer(dataJson),
	)

	if err != nil {
		fmt.Printf("failed to create request: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete note, status code: %d", resp.StatusCode)
	}

	return nil
}

func SyncNoteUpdate(
	vaultPath, oldFilePath, newFilePath string,
	s *state.State,
	claims *utils.Claims,
) error {
	noteContent, err := os.ReadFile(filepath.Join(vaultPath, newFilePath))
	if err != nil {
		return err
	}

	note := string(noteContent)
	frontMatter, content := splitFrontMatter(note)
	metadata, err := parseFrontMatter(frontMatter)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"new_title": strings.TrimSuffix(filepath.Base(newFilePath), ".md"),
		"title":     strings.TrimSuffix(filepath.Base(oldFilePath), ".md"),
		"tags":      metadata.Tags,
		"content":   content,
		"vault_id":  s.Config.VaultID,
		"user_id":   claims.UserID,
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return err
	}

	fmt.Println(string(dataJson))

	// Create the PATCH request
	req, err := http.NewRequest(
		"PATCH",
		"http://localhost:6474/v1/api/notes",
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		fmt.Printf("failed to create request: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send request: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Body)
		return fmt.Errorf("failed to update note, status code: %d", resp.StatusCode)
	}

	// Optionally decode the response data if needed
	var respData interface{}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("failed to decode response: %v", err)
		return err
	}

	fmt.Println(respData)

	return nil
}

// TODO: Our endpoint should probably update my title by default, rather than ID. We are more likely to have the matching title.
// TODO: In the event that a file changes names, we delete the old title record then create the new note record
func SyncNote(vaultPath, file string, s *state.State, claims *utils.Claims) error {
	noteContent, err := os.ReadFile(filepath.Join(vaultPath, file))
	if err != nil {
		return err
	}

	note := string(noteContent)
	frontMatter, content := splitFrontMatter(note)
	metadata, err := parseFrontMatter(frontMatter)
	if err != nil {
		return err
	}

	// TODO: Fix how LinkedNotes are sent, by title not ID
	// metadata.LinkedNotes = parseLinkedNotes(content)
	metadata.Upstream = strings.TrimPrefix(
		strings.TrimSuffix(metadata.Upstream, "]]"),
		"[[",
	)

	var title string

	if metadata.Title == "" {
		title = file
	} else {
		title = metadata.Title
	}

	data := map[string]interface{}{
		"title":    title,
		"tags":     metadata.Tags,
		"vault_id": s.Config.VaultID,
		"user_id":  claims.UserID,
		"content":  content,
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return err
	}

	req, err := http.NewRequest(
		"POST",
		"http://localhost:6474/v1/api/notes",
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		fmt.Printf("failed to create request: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Println("status code: ", resp.StatusCode)
		return nil
	}

	var respData interface{}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("failed to decode response: %v", err)
		return err
	}

	return nil
}

func splitFrontMatter(note string) (frontMatter, content string) {
	parts := strings.Split(note, "\n---\n")
	if len(parts) < 2 {
		return "", note
	}

	return strings.TrimSpace(parts[0]), parts[1]
}

func parseFrontMatter(frontMatter string) (Frontmatter, error) {
	var metadata Frontmatter
	err := yaml.Unmarshal([]byte(frontMatter), &metadata)
	if err != nil {
		return Frontmatter{}, fmt.Errorf("error parsing front matter: %v", err)
	}

	return metadata, nil
}

func parseLinkedNotes(content string) []string {
	pattern := regexp.MustCompile(`\[\[(.*?)\]\]`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	linkedNotes := make([]string, 0, len(matches))

	for _, match := range matches {
		linkedNotes = append(linkedNotes, match[1])
	}

	return linkedNotes
}
