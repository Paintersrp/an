package config

import (
	"reflect"
	"testing"
)

func TestFromFile(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromFile(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_ToFile(t *testing.T) {
	type fields struct {
		VaultDir     string
		Editor       string
		NvimArgs     string
		HomeDir      string
		Molecules    []string
		MoleculeMode string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				VaultDir:     tt.fields.VaultDir,
				Editor:       tt.fields.Editor,
				NvimArgs:     tt.fields.NvimArgs,
				HomeDir:      tt.fields.HomeDir,
				Molecules:    tt.fields.Molecules,
				MoleculeMode: tt.fields.MoleculeMode,
			}
			if err := cfg.ToFile(); (err != nil) != tt.wantErr {
				t.Errorf("Config.ToFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_AddMolecule(t *testing.T) {
	type fields struct {
		VaultDir     string
		Editor       string
		NvimArgs     string
		HomeDir      string
		Molecules    []string
		MoleculeMode string
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				VaultDir:     tt.fields.VaultDir,
				Editor:       tt.fields.Editor,
				NvimArgs:     tt.fields.NvimArgs,
				HomeDir:      tt.fields.HomeDir,
				Molecules:    tt.fields.Molecules,
				MoleculeMode: tt.fields.MoleculeMode,
			}
			cfg.AddMolecule(tt.args.name)
		})
	}
}

func TestConfig_GetConfigPath(t *testing.T) {
	type fields struct {
		VaultDir     string
		Editor       string
		NvimArgs     string
		HomeDir      string
		Molecules    []string
		MoleculeMode string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				VaultDir:     tt.fields.VaultDir,
				Editor:       tt.fields.Editor,
				NvimArgs:     tt.fields.NvimArgs,
				HomeDir:      tt.fields.HomeDir,
				Molecules:    tt.fields.Molecules,
				MoleculeMode: tt.fields.MoleculeMode,
			}
			if got := cfg.GetConfigPath(); got != tt.want {
				t.Errorf("Config.GetConfigPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_ChangeMode(t *testing.T) {
	type fields struct {
		VaultDir     string
		Editor       string
		NvimArgs     string
		HomeDir      string
		Molecules    []string
		MoleculeMode string
	}
	type args struct {
		mode string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				VaultDir:     tt.fields.VaultDir,
				Editor:       tt.fields.Editor,
				NvimArgs:     tt.fields.NvimArgs,
				HomeDir:      tt.fields.HomeDir,
				Molecules:    tt.fields.Molecules,
				MoleculeMode: tt.fields.MoleculeMode,
			}
			cfg.ChangeMode(tt.args.mode)
		})
	}
}

func TestConfig_ChangeEditor(t *testing.T) {
	type fields struct {
		VaultDir     string
		Editor       string
		NvimArgs     string
		HomeDir      string
		Molecules    []string
		MoleculeMode string
	}
	type args struct {
		editor string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				VaultDir:     tt.fields.VaultDir,
				Editor:       tt.fields.Editor,
				NvimArgs:     tt.fields.NvimArgs,
				HomeDir:      tt.fields.HomeDir,
				Molecules:    tt.fields.Molecules,
				MoleculeMode: tt.fields.MoleculeMode,
			}
			cfg.ChangeEditor(tt.args.editor)
		})
	}
}
