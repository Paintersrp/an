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
)

type NotePayload struct {
	Title    string   `json:"title"       validate:"required"`
	NewTitle string   `json:"new_title"`
	Tags     []string `json:"tags"`
	UserID   int32    `json:"user_id"     validate:"required"`
	VaultID  int32    `json:"vault_id"    validate:"required"`
	Upstream *int32   `json:"upstream_id"`
	Links    []int32  `json:"links"`
	Content  string   `json:"content"     validate:"required"`
}

type NoteDeletePayload struct {
	UserID int32  `json:"user_id" validate:"required"`
	Title  string `json:"title"   validate:"required"`
}

type NoteOperation struct {
	Operation     string             `json:"operation"`
	UpdatePayload *NotePayload       `json:"update_payload,omitempty"`
	DeletePayload *NoteDeletePayload `json:"delete_payload,omitempty"`
}

type BulkNoteOperationPayload struct {
	Operations []NoteOperation `json:"operations"`
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

	var operations []NoteOperation

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
			operations = append(operations, NoteOperation{
				Operation: "delete",
				DeletePayload: &NoteDeletePayload{
					Title:  strings.TrimSuffix(filepath.Base(oldFilePath), ".md"),
					UserID: int32(claims.UserID),
				},
			})
			fileDeleted = true
		}

		if !fileDeleted &&
			(filepath.Ext(oldFilePath) == ".md" || filepath.Ext(newFilePath) == ".md") {
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

			operations = append(operations, NoteOperation{
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
	}

	// Send the bulk operation payload to the server
	bulkPayload := BulkNoteOperationPayload{Operations: operations}
	dataJson, err := json.Marshal(bulkPayload)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return err
	}

	req, err := http.NewRequest(
		"POST",
		"http://localhost:6474/v1/api/notes/bulk",
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

	if resp.StatusCode != http.StatusOK {
		fmt.Println("status code: ", resp.StatusCode)
		return nil
	}

	var respData interface{}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("failed to decode response: %v", err)
		return err
	}

	fmt.Println(respData)

	return nil
}

func BulkSyncNotesInit(
	operations []NoteOperation,
	s *state.State,
	claims *utils.Claims,
) error {
	bulkPayload := BulkNoteOperationPayload{Operations: operations}
	dataJson, err := json.Marshal(bulkPayload)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return err
	}

	fmt.Println(bulkPayload)

	req, err := http.NewRequest(
		"POST",
		"http://localhost:6474/v1/api/notes/bulk",
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

	if resp.StatusCode != http.StatusOK {
		fmt.Println("status code: ", resp.StatusCode)
		return nil
	}

	return nil
}
