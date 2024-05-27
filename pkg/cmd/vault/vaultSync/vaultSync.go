package vaultSync

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/sync"
	"github.com/Paintersrp/an/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
)

// Hardcoded for now until SSH is rolling
var SECRET = "cPcCMY404opm1GTC2I9gwLOXBNhNVe9nNB++OhlY+0F0rZ4LpJwhmFLEnzlSWupdbxzZjRDUlIfZk96bsv+gWg=="

type Frontmatter struct {
	Title       string   `yaml:"title"`
	Upstream    string   `yaml:"up"`
	CreatedAt   string   `yaml:"created"`
	Tags        []string `yaml:"tags"`
	LinkedNotes []string
}

type RemoteChange struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	NoteID    int       `json:"note_id"`
	Action    string    `json:"action"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Processed bool      `json:"processed"`
}

type ChangesPayload struct {
	Notes []RemoteChange `json:"notes"`
	// Tags  []RemoteTagChange `json:"tags"`
	// Links []RemoteLinkChange `json:"links"`
}

func NewCmdVaultSync(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync notes with the latest changes",
		Long:  "Sync notes with the latest changes in the repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			claims, err := utils.GetClaims(s.Config.Token, SECRET)
			cobra.CheckErr(err)

			rootDir := s.Config.RootDir
			vaultPath := filepath.Join(rootDir, s.Config.ActiveVault)

			repo, err := git.PlainOpen(vaultPath)
			if err != nil {
				return err
			}

			backupDir := filepath.Join(vaultPath, "backup")
			err = os.MkdirAll(backupDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create backup directory: %v", err)
			}
			defer os.RemoveAll(backupDir)

			changes, err := FetchRemoteChanges(s)
			if err != nil {
				return err
			}

			var modifiedFiles []string
			if len(changes.Notes) > 0 {
				modifiedFiles, err = ApplyRemoteChanges(
					vaultPath,
					backupDir,
					repo,
					changes.Notes,
					s,
					claims,
				)
				if err != nil {
					fmt.Println("err - ApplyRemoteChanges")
					return err
				}
			}

			if len(modifiedFiles) > 0 {
				commit, err := CommitChanges(
					repo,
					modifiedFiles,
					claims,
					"Apply remote changes",
				)

				if err != nil {
					rollback(backupDir, modifiedFiles)
					return fmt.Errorf("failed to commit changes: %v", err)
				}

				// TODO: ARE WE CREATING THE NOTES ON THE REMOTE CREATE OR JUST THE REMOTE RECORD?
				// syncErr := sync.BulkSyncNotes(vaultPath, repo, commit, s, claims)
				// if syncErr != nil {
				// 	return syncErr
				// }

				err = PostProcessedChanges(s)
				if err != nil {
					rollback(backupDir, modifiedFiles)
					return fmt.Errorf("failed to post-process changes: %v", err)
				}

				err = sync.UpdateVaultCommit(commit.String(), s)
				if err != nil {
					rollback(backupDir, modifiedFiles)
					return fmt.Errorf("failed to update vault commit: %v", err)
				}
			}

			worktree, err := repo.Worktree()
			if err != nil {
				return err
			}

			_, err = worktree.Add(".")
			if err != nil {
				return err
			}

			status, err := worktree.Status()
			if err != nil {
				return err
			}

			if !status.IsClean() {
				commit, err := CommitAllChanges(repo, claims, "Sync local changes")
				if err != nil {
					return err
				}

				syncErr := sync.BulkSyncNotes(vaultPath, repo, commit, s, claims)
				if syncErr != nil {
					return syncErr
				}

				err = sync.UpdateVaultCommit(commit.String(), s)
				if err != nil {
					return err
				}
			} else {
				cmd.Println("No additional changes to sync. The vault is up-to-date.")
			}

			return nil
		},
	}

	return cmd
}

func FetchRemoteChanges(s *state.State) (ChangesPayload, error) {
	var payload ChangesPayload

	url := "http://localhost:6474/v1/api/notes/remote"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return payload, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return payload, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return payload, fmt.Errorf("failed to fetch remote changes: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return payload, err
	}

	return payload, nil
}

func PostProcessedChanges(s *state.State) error {
	url := "http://localhost:6474/v1/api/notes/remote/process"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Config.Token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch remote changes: %s", resp.Status)
	}

	return nil

}

func ApplyRemoteChanges(
	vaultPath string,
	backupDir string,
	repo *git.Repository,
	changes []RemoteChange,
	s *state.State,
	claims *utils.Claims,
) ([]string, error) {
	var modifiedFiles []string

	for _, change := range changes {
		notePath := filepath.Join(vaultPath, fmt.Sprintf("%s.md", change.Title))

		switch change.Action {
		case "create":
			fmt.Println("creating...")
			// TODO: Remove hardcoded subdir
			notePath = filepath.Join(
				vaultPath,
				"atoms",
				fmt.Sprintf("%s.md", change.Title),
			)
			if err := os.WriteFile(notePath, []byte(change.Content), 0644); err != nil {
				return nil, fmt.Errorf("failed to create note: %v", err)
			}
			modifiedFiles = append(modifiedFiles, notePath)

		case "update":
			if fileModifiedLocally(notePath, repo) {
				_, err := os.ReadFile(notePath)
				if err != nil {
					return nil, fmt.Errorf("failed to read local note: %v", err)
				}

				backupPath := filepath.Join(backupDir, filepath.Base(notePath))
				err = copyFile(notePath, backupPath)
				if err != nil {
					return nil, fmt.Errorf("failed to copy note: %v", err)
				}

				if change.CreatedAt.After(getLocalFileModTime(notePath)) {
					if err := os.WriteFile(notePath, []byte(change.Content), 0644); err != nil {
						return nil, fmt.Errorf("failed to update note: %v", err)
					}
					modifiedFiles = append(modifiedFiles, notePath)
				} else {
					// TODO: Conflict handling
					fmt.Printf("Conflict detected for note %s. Keeping local changes.\n", change.Title)
				}
			} else {
				backupPath := filepath.Join(backupDir, filepath.Base(notePath))
				err := copyFile(notePath, backupPath)
				if err != nil {
					return nil, fmt.Errorf("failed to copy note: %v", err)
				}

				if err := os.WriteFile(notePath, []byte(change.Content), 0644); err != nil {
					return nil, fmt.Errorf("failed to update note: %v", err)
				}
				modifiedFiles = append(modifiedFiles, notePath)
			}

		case "delete":
			if fileModifiedLocally(notePath, repo) {
				// TODO: Conflict handling
				fmt.Printf(
					"Conflict detected for note %s. Keeping local changes.\n",
					change.Title,
				)
			} else {
				backupPath := filepath.Join(backupDir, filepath.Base(notePath))
				err := copyFile(notePath, backupPath)
				if err != nil {
					return nil, fmt.Errorf("failed to copy note: %v", err)
				}

				if err := os.Remove(notePath); err != nil {
					return nil, fmt.Errorf("failed to delete note: %v", err)
				}
				modifiedFiles = append(modifiedFiles, notePath)
			}
		default:
			return nil, fmt.Errorf("invalid action: %s", change.Action)
		}
	}

	return modifiedFiles, nil
}

func CommitChanges(
	repo *git.Repository,
	files []string,
	claims *utils.Claims,
	message string,
) (plumbing.Hash, error) {
	var commit plumbing.Hash

	worktree, err := repo.Worktree()
	if err != nil {
		fmt.Println("worktree err")
		return commit, err
	}

	repoPath := worktree.Filesystem.Root()

	for _, file := range files {
		relPath, err := filepath.Rel(repoPath, file)
		if err != nil {
			fmt.Println("error computing relative path")
			return commit, err
		}
		fmt.Println(relPath)
		_, err = worktree.Add(relPath)
		if err != nil {
			fmt.Println("file error")
			return commit, err
		}
	}

	commit, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  claims.Username,
			Email: claims.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return commit, err
	}

	return commit, nil
}

func CommitAllChanges(
	repo *git.Repository,
	claims *utils.Claims,
	message string,
) (plumbing.Hash, error) {
	var commit plumbing.Hash

	worktree, err := repo.Worktree()
	if err != nil {
		return commit, err
	}

	_, err = worktree.Add(".")
	if err != nil {
		return commit, err
	}

	commit, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  claims.Username,
			Email: claims.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return commit, err
	}

	return commit, nil
}

func fileModifiedLocally(filePath string, repo *git.Repository) bool {
	wt, err := repo.Worktree()
	if err != nil {
		return false
	}
	status, err := wt.Status()
	if err != nil {
		return false
	}
	return status.IsUntracked(filePath) ||
		status.File(filePath).Worktree == git.Unmodified
}

func getLocalFileModTime(filePath string) time.Time {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}
	}
	return fileInfo.ModTime()
}

func rollback(backupDir string, modifiedFiles []string) {
	for _, file := range modifiedFiles {
		os.Remove(file)
	}

	files, _ := os.ReadDir(backupDir)
	for _, file := range files {
		backupPath := filepath.Join(backupDir, file.Name())
		targetPath := filepath.Join(filepath.Dir(backupDir), file.Name())
		copyFile(backupPath, targetPath)
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}
	err = os.Chmod(dst, info.Mode())
	if err != nil {
		return err
	}

	return nil
}
