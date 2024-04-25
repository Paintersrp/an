/*
Copyright © 2024 Ryan Painter paintersrp@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/constants"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	moleculeName string

	appCfg   *config.Config
	cfgError error

	appTemplater   *templater.Templater
	templaterError error
)

var rootCmd = &cobra.Command{
	Use:     "atomic",
	Aliases: []string{"an", "a-n"},
	Short:   "Launch into writing atomic notes, blended into integration with Obsidian.",
	Long: `A utility to help you get into the habit of writing notes by providing ways to quickly
  get up and writing with atomic notes. 

              [title]  [tags]
  an new robotics "robotics science class study-notes"
  `,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(ensureConfigExists, initConfig)

	rootCmd.PersistentFlags().
		StringVarP(&moleculeName, "molecule", "m", "atoms", "Molecule subdirectory to use for this command.")
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.an/cfg.yaml)")
	viper.BindPFlag(
		"molecule",
		rootCmd.PersistentFlags().Lookup("molecule"),
	)

	appTemplater, templaterError = templater.NewTemplater()

	if templaterError != nil {
		fmt.Printf(
			"failed to create templater: %v",
			templaterError,
		)
		cobra.CheckErr(templaterError)
	}

	// Validate the molecule flag
	cobra.OnInitialize(handleMolecules)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home + constants.ConfigDir)
		viper.SetConfigName(constants.ConfigFile)
		viper.SetConfigType(constants.ConfigFileType)
	}

	// If a config file is found, read it in.
	viper.ReadInConfig()
	appCfg, cfgError = config.FromFile(
		viper.ConfigFileUsed(),
	)
}

func ensureConfigExists() {
	home, homeErr := os.UserHomeDir()
	cobra.CheckErr(homeErr)

	// Get the directory path of the file and absolute file path
	dir := fmt.Sprintf("%s/%s", home, constants.ConfigDir)
	filePath := fmt.Sprintf(
		"%s/%s.%s",
		dir,
		constants.ConfigFile,
		constants.ConfigFileType,
	)

	// Check if the directory already exists
	_, dirErr := os.Stat(dir)
	if os.IsNotExist(dirErr) {
		// If the directory does not exist, create it
		err := os.MkdirAll(dir, os.ModePerm)
		cobra.CheckErr(err)
	}

	// Check if the file already exists
	_, fileErr := os.Stat(filePath)
	if os.IsNotExist(fileErr) {
		// If the file does not exist, create an empty file
		file, err := os.Create(filePath)
		cobra.CheckErr(err)
		file.Close()
	}
}

func handleMolecules() {
	mode := viper.GetString("moleculeMode")
	exists, err := verifyMoleculeExists()
	cobra.CheckErr(err)
	switch mode {
	case "strict":
		if !exists {
			fmt.Println(
				"Error: Molecule",
				moleculeName,
				"does not exist.",
			)
			fmt.Println(
				"In strict mode, new molecules are added with the add-molecule command.",
			)
			os.Exit(1)
		}
	case "free":
		if !exists {
			AddMoleculeToConfig(moleculeName)
		}
	case "confirm":
		if !exists {
			getConfirmation()
		}
	default:
		if !exists {
			getConfirmation()
		}
	}
}

func verifyMoleculeExists() (bool, error) {
	var molecules []string
	if err := viper.UnmarshalKey("molecules", &molecules); err != nil {
		fmt.Println("Error unmarshalling molecules:", err)
		return false, err
	}

	// Check if the specified molecule exists
	moleculeExists := false
	for _, molecule := range molecules {
		if molecule == moleculeName {
			moleculeExists = true
			break
		}
	}

	if !moleculeExists {
		return moleculeExists, nil
	}

	return moleculeExists, nil
}

func getConfirmation() {
	var response string
	for {
		fmt.Printf(
			"Molecule %s does not exist.\nDo you want to create it?\n(y/n): ",
			moleculeName,
		)
		fmt.Scanln(&response)
		response = strings.ToLower(
			strings.TrimSpace(response),
		)

		switch response {
		case "yes", "y":
			// TODO: should use appCfg.AddMolecule
			AddMoleculeToConfig(moleculeName)
			return
		case "no", "n":
			fmt.Println(
				"Exiting due to non-existing molecule",
			)
			os.Exit(0)
		default:
			fmt.Println(
				"Invalid response. Please enter 'y'/'yes' or 'n'/'no'.",
			)
		}
	}
}
