/*
Copyright © 2024 Ryan Painter <paintersrp@gmail.com>
*/

package cmd

import "testing"

func TestAddMoleculeToConfig(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddMoleculeToConfig(tt.args.name)
		})
	}
}
