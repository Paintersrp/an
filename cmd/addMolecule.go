/*
Copyright © 2024 Ryan Painter <paintersrp@gmail.com>
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var addMoleculeCmd = &cobra.Command{
	Use:     "add-molecule [name]",
	Aliases: []string{"am", "add-m", "a-m"},
	Short:   "Add a molecule to the list of available projects.",
	Long:    `This command adds a molecule to the list of available projects in the global persistent configuration.`,
	Example: "atomic add-molecule Go-CLI",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		appCfg.AddMolecule(name)
		// AddMoleculeToConfig(name)
	},
}

// Initialize the add-molecule command
func init() {
	rootCmd.AddCommand(addMoleculeCmd)
}

func AddMoleculeToConfig(name string) {
	// Get existing molecules from the configuration
	var molecules []string
	if err := viper.UnmarshalKey("molecules", &molecules); err != nil {
		fmt.Println("Error unmarshalling molecules:", err)
		return
	}

	// Check if the molecule already exists
	for _, molecule := range molecules {
		if molecule == name {
			fmt.Println("Molecule", name, "already exists.")
			return
		}
	}

	// Append the new molecule
	molecules = append(molecules, name)

	// Store the updated molecules in the configuration
	viper.Set("molecules", molecules)

	// Save the configuration
	if err := viper.WriteConfig(); err != nil {
		fmt.Println("Error writing configuration file:", err)
		return
	}

	fmt.Println("Molecule", name, "added successfully.")
}
