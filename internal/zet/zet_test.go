// Package zet provides functionality for managing zettelkasten (atomic) notes.
package zet

import (
	"reflect"
	"testing"

	"github.com/Paintersrp/an/internal/templater"
)

func TestZettelkastenNote_GetFilepath(t *testing.T) {
	type fields struct {
		VaultDir     string
		SubDir       string
		Filename     string
		OriginalTags []string
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
			note := &ZettelkastenNote{
				VaultDir:     tt.fields.VaultDir,
				SubDir:       tt.fields.SubDir,
				Filename:     tt.fields.Filename,
				OriginalTags: tt.fields.OriginalTags,
			}
			if got := note.GetFilepath(); got != tt.want {
				t.Errorf("ZettelkastenNote.GetFilepath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZettelkastenNote_EnsurePath(t *testing.T) {
	type fields struct {
		VaultDir     string
		SubDir       string
		Filename     string
		OriginalTags []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &ZettelkastenNote{
				VaultDir:     tt.fields.VaultDir,
				SubDir:       tt.fields.SubDir,
				Filename:     tt.fields.Filename,
				OriginalTags: tt.fields.OriginalTags,
			}
			got, err := note.EnsurePath()
			if (err != nil) != tt.wantErr {
				t.Errorf("ZettelkastenNote.EnsurePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ZettelkastenNote.EnsurePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZettelkastenNote_FileExists(t *testing.T) {
	type fields struct {
		VaultDir     string
		SubDir       string
		Filename     string
		OriginalTags []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &ZettelkastenNote{
				VaultDir:     tt.fields.VaultDir,
				SubDir:       tt.fields.SubDir,
				Filename:     tt.fields.Filename,
				OriginalTags: tt.fields.OriginalTags,
			}
			got, got1, err := note.FileExists()
			if (err != nil) != tt.wantErr {
				t.Errorf("ZettelkastenNote.FileExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ZettelkastenNote.FileExists() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ZettelkastenNote.FileExists() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestZettelkastenNote_Create(t *testing.T) {
	type fields struct {
		VaultDir     string
		SubDir       string
		Filename     string
		OriginalTags []string
	}
	type args struct {
		tmplName string
		t        *templater.Templater
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &ZettelkastenNote{
				VaultDir:     tt.fields.VaultDir,
				SubDir:       tt.fields.SubDir,
				Filename:     tt.fields.Filename,
				OriginalTags: tt.fields.OriginalTags,
			}
			got, err := note.Create(tt.args.tmplName, tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("ZettelkastenNote.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ZettelkastenNote.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZettelkastenNote_Open(t *testing.T) {
	type fields struct {
		VaultDir     string
		SubDir       string
		Filename     string
		OriginalTags []string
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
			note := &ZettelkastenNote{
				VaultDir:     tt.fields.VaultDir,
				SubDir:       tt.fields.SubDir,
				Filename:     tt.fields.Filename,
				OriginalTags: tt.fields.OriginalTags,
			}
			if err := note.Open(); (err != nil) != tt.wantErr {
				t.Errorf("ZettelkastenNote.Open() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewZettelkastenNote(t *testing.T) {
	type args struct {
		vaultDir string
		subDir   string
		filename string
		tags     []string
	}
	tests := []struct {
		name string
		args args
		want *ZettelkastenNote
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewZettelkastenNote(tt.args.vaultDir, tt.args.subDir, tt.args.filename, tt.args.tags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewZettelkastenNote() = %v, want %v", got, tt.want)
			}
		})
	}
}
