package vaultSync

import (
	"path/filepath"
	"time"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/sync"
	"github.com/Paintersrp/an/utils"
	"github.com/go-git/go-git/v5"
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

func NewCmdVaultSync(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync notes with the latest changes",
		Long:  "Sync notes with the latest changes in the Git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			claims, err := utils.GetClaims(s.Config.Token, SECRET)
			cobra.CheckErr(err)

			// vault, err := utils.GetVaultByName(
			// 	s.Config.Token,
			// 	s.Config.ActiveVault,
			// )
			// if err != nil {
			// 	return err
			// }

			rootDir := s.Config.RootDir
			vaultPath := filepath.Join(rootDir, s.Config.ActiveVault)

			repo, err := git.PlainOpen(vaultPath)
			if err != nil {
				return err
			}

			worktree, err := repo.Worktree()
			if err != nil {
				return err
			}

			// Stage all changes (add, modify, remove)
			_, err = worktree.Add(".")
			if err != nil {
				return err
			}

			status, err := worktree.Status()
			if err != nil {
				return err
			}

			if status.IsClean() {
				cmd.Println("No changes to sync. The vault is up-to-date.")
				return nil
			}

			commit, err := worktree.Commit("Sync changes", &git.CommitOptions{
				Author: &object.Signature{
					Name:  claims.Username,
					Email: claims.Email,
					When:  time.Now(),
				},
			})
			if err != nil {
				return err
			}

			syncErr := sync.SyncNotes(vaultPath, repo, commit, s, claims)
			if syncErr != nil {
				return syncErr
			}

			// Update the vault's commit hash after successful sync
			err = sync.UpdateVaultCommit(commit.String(), s)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
