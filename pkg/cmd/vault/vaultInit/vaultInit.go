package vaultInit

import (
	"fmt"
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

var SECRET = "cPcCMY404opm1GTC2I9gwLOXBNhNVe9nNB++OhlY+0F0rZ4LpJwhmFLEnzlSWupdbxzZjRDUlIfZk96bsv+gWg=="

func NewCmdVaultInit(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new vault",
		Long:  "Initialize a new vault by creating a Git repository and committing existing Markdown files.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			claims, err := utils.GetClaims(s.Config.Token, SECRET)
			if err != nil {
				return err
			}

			name := args[0]

			nameErr := utils.IsValidDirName(name)
			if nameErr != nil {
				cmd.Println("Error:", nameErr)
				return nameErr
			}

			vault, err := utils.SendVaultCreateRequest(s.Config.Token, name, claims)
			if err != nil {
				return err
			}
			fmt.Println(vault.ID, vault.Name, "vault")

			cfgErr := s.Config.ChangeVault(name, vault.ID)
			if cfgErr != nil {
				return cfgErr
			}

			rootDir := s.Config.RootDir
			vaultPath := filepath.Join(rootDir, name)

			// Check if the directory already exists
			_, dirErr := os.Stat(vaultPath)
			if dirErr != nil {
				if os.IsNotExist(dirErr) {
					cmd.Printf("Directory %s does not exist\n", name)
					return dirErr
				}
				cmd.Println("Error:", dirErr)
				return dirErr
			}

			// Initialize Git repository
			repo, err := git.PlainInit(vaultPath, false)
			if err != nil {
				cmd.Println("Error:", err)
				return err
			}

			// Commit existing Markdown files
			worktree, err := repo.Worktree()
			if err != nil {
				cmd.Println("Error:", err)
				return err
			}

			commit, err := commitMarkdownFiles(worktree, vaultPath, claims)
			if err != nil {
				cmd.Println("Error:", err)
				return err
			}

			// Sync notes after the commit
			err = sync.SyncNotesInit(vaultPath, repo, commit, s, claims)
			if err != nil {
				cmd.Println("Error:", err)
				return err
			}

			// Update vault commit
			err = sync.UpdateVaultCommit(commit.String(), s)
			if err != nil {
				return err
			}

			cmd.Printf("Vault %s initialized successfully\n", name)
			return nil
		},
	}

	return cmd
}

func commitMarkdownFiles(
	worktree *git.Worktree,
	vaultPath string,
	claims *utils.Claims,
) (plumbing.Hash, error) {
	// Create .gitignore file
	gitignorePath := filepath.Join(vaultPath, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("*\n!*.md"), 0644)
	if err != nil {
		return plumbing.Hash{}, err
	}

	_, addErr := worktree.Add(".")
	if addErr != nil {
		return plumbing.Hash{}, addErr
	}

	commit, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  claims.Username,
			Email: claims.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return plumbing.Hash{}, err
	}

	return commit, err
}
