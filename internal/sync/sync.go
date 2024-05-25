package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TODO: Tags are being added in but not removed correctly
// TODO: Linked Notes still need handling
// TODO: General Cleanup

// TODO: Fix how LinkedNotes are sent, by title not ID
// metadata.LinkedNotes = parseLinkedNotes(content)

func BulkRequest(
	operations []NoteOperation,
	s *state.State,
	claims *utils.Claims,
) error {
	bulkPayload := BulkNoteOperationPayload{Operations: operations}
	dataJson, err := json.Marshal(bulkPayload)
	if err != nil {
		return fmt.Errorf("failed to encode data to JSON: %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		"http://localhost:6474/v1/api/notes/bulk",
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
		return fmt.Errorf("failed to get commit object: %v", err)
	}

	tree, err := commitObj.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree from commit: %v", err)
	}

	files, err := getMarkdownFiles(tree)
	if err != nil {
		return fmt.Errorf("failed to get markdown files: %v", err)
	}

	operations, err := prepareNoteOperations(vaultPath, files, s, claims)
	if err != nil {
		return err
	}

	return BulkRequest(operations, s, claims)
}

func prepareNoteOperations(
	vaultPath string,
	files []string,
	s *state.State,
	claims *utils.Claims,
) ([]NoteOperation, error) {
	var operations []NoteOperation

	for _, file := range files {
		noteContent, err := os.ReadFile(filepath.Join(vaultPath, file))
		if err != nil {
			fmt.Printf("failed to read file %s: %v\n", file, err)
			continue
		}

		if noteContent == nil {
			continue
		}

		note := string(noteContent)
		frontMatter, content := splitFrontMatter(note)
		metadata, err := parseFrontMatter(frontMatter)
		if err != nil {
			fmt.Printf("failed to parse frontmatter for %s: %v\n", file, err)
			continue
		}

		if content == "" {
			continue
		}

		metadata.Upstream = strings.TrimPrefix(
			strings.TrimSuffix(metadata.Upstream, "]]"),
			"[[",
		)

		operations = append(operations, NoteOperation{
			Operation: "create",
			UpdatePayload: &NotePayload{
				Title:   strings.TrimSuffix(filepath.Base(file), ".md"),
				Tags:    metadata.Tags,
				Content: content,
				VaultID: s.Config.VaultID,
				UserID:  int32(claims.UserID),
			},
		})
	}

	return operations, nil
}

func BulkSyncNotes(
	vaultPath string,
	repo *git.Repository,
	commit plumbing.Hash,
	s *state.State,
	claims *utils.Claims,
) error {
	currentCommitObj, err := repo.CommitObject(commit)
	if err != nil {
		return fmt.Errorf("failed to get current commit object: %v", err)
	}

	parents := currentCommitObj.Parents()
	defer parents.Close()
	parentCommitObj, err := parents.Next()
	if err != nil {
		return fmt.Errorf("failed to get parent commit object: %v", err)
	}

	patch, err := parentCommitObj.Patch(currentCommitObj)
	if err != nil {
		return fmt.Errorf("failed to get patch: %v", err)
	}

	operations, err := preparePatchOperations(vaultPath, patch, s, claims)
	if err != nil {
		return err
	}

	return BulkRequest(operations, s, claims)
}

func preparePatchOperations(
	vaultPath string,
	patch *object.Patch,
	s *state.State,
	claims *utils.Claims,
) ([]NoteOperation, error) {
	var operations []NoteOperation

	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()

		switch {
		case from != nil && to != nil:
			// File was modified
			oldFilePath := from.Path()
			newFilePath := to.Path()

			if err := handleModifiedFile(vaultPath, oldFilePath, newFilePath, &operations, s, claims); err != nil {
				return nil, err
			}

		case to != nil:
			// File was added
			newFilePath := to.Path()

			if err := handleCreatedFile(vaultPath, newFilePath, &operations, s, claims); err != nil {
				return nil, err
			}

		case from != nil:
			// File was deleted
			oldFilePath := from.Path()
			handleDeletedFile(oldFilePath, &operations, claims)
		}
	}

	return operations, nil
}

func handleModifiedFile(
	vaultPath, oldFilePath, newFilePath string,
	operations *[]NoteOperation,
	s *state.State,
	claims *utils.Claims,
) error {
	if filepath.Ext(oldFilePath) == ".md" || filepath.Ext(newFilePath) == ".md" {
		metadata, content, err := parseNote(vaultPath, newFilePath)
		if err != nil {
			return err
		}

		*operations = append(*operations, NoteOperation{
			Operation: "update",
			UpdatePayload: &NotePayload{
				NewTitle: strings.TrimSuffix(filepath.Base(newFilePath), ".md"),
				Title:    strings.TrimSuffix(filepath.Base(oldFilePath), ".md"),
				Tags:     metadata.Tags,
				Content:  content,
				VaultID:  s.Config.VaultID,
				UserID:   int32(claims.UserID),
			},
		})
	}

	return nil
}

func handleCreatedFile(
	vaultPath, newFilePath string,
	operations *[]NoteOperation,
	s *state.State,
	claims *utils.Claims,
) error {
	metadata, content, err := parseNote(vaultPath, newFilePath)
	if err != nil {
		return err
	}

	if filepath.Ext(newFilePath) == ".md" {
		*operations = append(*operations, NoteOperation{
			Operation: "create",
			UpdatePayload: &NotePayload{
				Title:   strings.TrimSuffix(filepath.Base(newFilePath), ".md"),
				Tags:    metadata.Tags,
				Content: content,
				VaultID: s.Config.VaultID,
				UserID:  int32(claims.UserID),
			},
		})
	}

	return nil
}

func handleDeletedFile(
	oldFilePath string,
	operations *[]NoteOperation,
	claims *utils.Claims,
) {
	*operations = append(*operations, NoteOperation{
		Operation: "delete",
		DeletePayload: &NoteDeletePayload{
			Title:  strings.TrimSuffix(filepath.Base(oldFilePath), ".md"),
			UserID: int32(claims.UserID),
		},
	})
}

func parseNote(vaultPath, path string) (Frontmatter, string, error) {
	noteContent, err := os.ReadFile(filepath.Join(vaultPath, path))
	if err != nil {
		return Frontmatter{}, "", fmt.Errorf(
			"failed to read modified file %s: %v",
			path,
			err,
		)
	}

	note := string(noteContent)
	frontMatter, content := splitFrontMatter(note)
	metadata, err := parseFrontMatter(frontMatter)

	if err != nil {
		return Frontmatter{}, "", fmt.Errorf(
			"failed to parse frontmatter for %s: %v",
			path,
			err,
		)
	}

	return metadata, content, nil
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
		return fmt.Errorf("failed to encode data to JSON: %v", err)
	}

	req, err := http.NewRequest(
		"PATCH",
		fmt.Sprintf("http://localhost:6474/v1/api/vaults/%d", s.Config.VaultID),
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
