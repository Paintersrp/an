package vaultAdd

import (
	"os"
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

func NewCmdVaultAdd(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new vault",
		Long:  "Add a new vault by creating a new directory and initializing a Git repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			claims, err := utils.GetClaims(s.Config.Token, SECRET)
			if err != nil {
				return err
			}

			name := args[0]

			if err := utils.IsValidDirName(name); err != nil {
				cmd.Println("Error:", err)
				return err
			}

			rootDir := s.Config.RootDir
			vaultPath := filepath.Join(rootDir, name)

			// Check if the directory already exists
			_, dirErr := os.Stat(vaultPath)
			if dirErr == nil {
				cmd.Printf("Directory %s already exists\n", name)
				return dirErr
			} else if !os.IsNotExist(dirErr) {
				cmd.Println("Error:", dirErr)
				return dirErr
			}

			// Create the new directory
			err = os.Mkdir(vaultPath, 0755)
			if err != nil {
				cmd.Println("Error:", err)
				return err
			}

			// Initialize Git repository and commit Markdown files
			err = initVaultRepo(s, name, vaultPath, claims)
			if err != nil {
				cmd.Println("Error:", err)
				return err
			}

			cmd.Printf("Vault %s created and initialized successfully\n", name)

			return nil
		},
	}

	return cmd
}

// TODO: Revert / Atomic Logic
func initVaultRepo(
	s *state.State,
	name string,
	vaultPath string,
	claims *utils.Claims,
) error {
	vault, err := utils.SendVaultCreateRequest(s.Config.Token, name, claims)
	if err != nil {
		return err
	}

	cfgErr := s.Config.ChangeVault(name, vault.ID)
	if cfgErr != nil {
		return cfgErr
	}

	repo, err := git.PlainInit(vaultPath, false)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	gitignorePath := filepath.Join(vaultPath, ".gitignore")
	err = os.WriteFile(gitignorePath, []byte("*\n!*.md"), 0644)
	if err != nil {
		return err
	}

	initialNotePath := filepath.Join(vaultPath, "First Note.md")
	err = os.WriteFile(
		initialNotePath,
		[]byte("---\ntitle: First Note\n---\n# First Note"),
		0644,
	)
	if err != nil {
		return err
	}

	_, err = worktree.Add(".")
	if err != nil {
		return err
	}

	commit, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  claims.Username,
			Email: claims.Email,
			When:  time.Now(),
		},
	})

	if err != nil {
		return err
	}

	// Sync notes after the commit
	err = sync.BulkSyncNotes(vaultPath, repo, commit, s, claims)
	if err != nil {
		return err
	}

	// Update vault commit
	err = sync.UpdateVaultCommit(commit.String(), s)
	if err != nil {
		return err
	}

	return err
}
