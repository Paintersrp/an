package vaultChange

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/utils"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

func NewCmdVaultChange(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "change [name]",
		Aliases: []string{"c"},
		Short:   "",
		Long:    heredoc.Doc(``),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := utils.IsValidDirName(name); err != nil {
				cmd.Println("Error:", err)
				return err
			}

			// Check if the specified vault exists
			rootDir := s.Config.RootDir
			vaultPath := filepath.Join(rootDir, name)
			vaultExists, err := vaultExists(vaultPath)
			if err != nil {
				cmd.Println("Error:", err)
				return err
			}

			if !vaultExists {
				cmd.Println("the specified vault does not exist.")
				return errors.New("to create a new vault, run `an vault add [name]`")
			}

			record, err := getVaultRecordByTitle(s.Config.Token, name)
			if err != nil {
				return err
			}

			cfgErr := s.Config.ChangeVault(name, record.ID)
			if cfgErr != nil {
				return cfgErr
			}

			cmd.Printf("Successfully changed to vault: %s", vaultPath)
			return nil
		},
	}

	return cmd
}

func vaultExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if !fi.IsDir() {
		return false, fmt.Errorf("%s is not a directory", path)
	}

	// Check if the directory is a Git repository
	_, err = git.PlainOpen(path)
	if err == git.ErrRepositoryNotExists {
		return false, err
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func getVaultRecordByTitle(
	token,
	name string,
) (*utils.VaultRecord, error) {
	data := map[string]interface{}{
		"name": name,
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		"http://localhost:6474/v1/api/vaults/find",
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		fmt.Printf("failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var respData utils.VaultRecord
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("failed to decode response: %v", err)
		return nil, err
	}

	return &respData, nil
}
