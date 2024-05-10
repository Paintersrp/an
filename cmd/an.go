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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/fs/templater"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/constants"
	"github.com/Paintersrp/an/pkg/cmd/root"
)

func Execute() {
	// Get Home Directory for locating config files
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	// Eventually will factor out Viper entirely
	viper.AddConfigPath(home + constants.ConfigDir)
	viper.SetConfigName(constants.ConfigFile)
	viper.SetConfigType(constants.ConfigFileType)
	viper.ReadInConfig()

	config.EnsureConfigExists(home)
	cfg, cfgErr := config.FromFile(config.StaticGetConfigPath(home))
	if cfgErr != nil {
		return // exit?
	}

	templater, templaterErr := templater.NewTemplater()
	if templaterErr != nil {
		fmt.Printf("failed to create templater: %v", templaterErr)
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
