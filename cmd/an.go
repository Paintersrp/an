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

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/constants"
	"github.com/Paintersrp/an/pkg/cmd/root"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() {
	cobra.OnInitialize(ensureConfigExists)
	cfg, cfgErr := initConfig()
	if cfgErr != nil {
		return // exit?
	}

	templater, templaterErr := templater.NewTemplater()

	if templaterErr != nil {
		fmt.Printf(
			"failed to create templater: %v",
			templaterErr,
		)
		cobra.CheckErr(templaterErr)
	}
	rootCmd, rootErr := root.NewCmdRoot(cfg, templater)
	if rootErr != nil {
		return // exit?
	}

	execErr := rootCmd.Execute()
	if execErr != nil {
		os.Exit(1)
	}
}

func initConfig() (*config.Config, error) {
	// Find home directory.
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	viper.AddConfigPath(home + constants.ConfigDir)
	viper.SetConfigName(constants.ConfigFile)
	viper.SetConfigType(constants.ConfigFileType)

	// If a config file is found, read it in.
	viper.ReadInConfig()

	cfg, err := config.FromFile(
		config.StaticGetConfigPath(home),
	)

	return cfg, err
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
