package fzf

import (
	"reflect"
	"testing"
)

func TestNewFuzzyFinder(t *testing.T) {
	type args struct {
		vaultDir string
	}
	tests := []struct {
		name string
		args args
		want *FuzzyFinder
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFuzzyFinder(tt.args.vaultDir); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf(
					"NewFuzzyFinder() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestFuzzyFinder_Run(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			f.Run()
		})
	}
}

func TestFuzzyFinder_RunWithQuery(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	type args struct {
		query string
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
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			f.RunWithQuery(tt.args.query)
		})
	}
}

func TestFuzzyFinder_findAndExecute(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	type args struct {
		query string
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
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			f.findAndExecute(tt.args.query)
		})
	}
}

func TestFuzzyFinder_listFiles(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			got, err := f.listFiles()
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"FuzzyFinder.listFiles() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"FuzzyFinder.listFiles() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestFuzzyFinder_fuzzySelectFile(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	type args struct {
		query string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			got, err := f.fuzzySelectFile(tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"FuzzyFinder.fuzzySelectFile() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
				return
			}
			if got != tt.want {
				t.Errorf(
					"FuzzyFinder.fuzzySelectFile() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestFuzzyFinder_renderMarkdownPreview(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	type args struct {
		i int
		w int
		h int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			if got := f.renderMarkdownPreview(tt.args.i, tt.args.w, tt.args.h); got != tt.want {
				t.Errorf(
					"FuzzyFinder.renderMarkdownPreview() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func Test_parseFrontMatter(t *testing.T) {
	type args struct {
		content []byte
	}
	tests := []struct {
		name      string
		args      args
		wantTitle string
		wantTags  []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotTags := parseFrontMatter(tt.args.content)
			if gotTitle != tt.wantTitle {
				t.Errorf(
					"parseFrontMatter() gotTitle = %v, want %v",
					gotTitle,
					tt.wantTitle,
				)
			}
			if !reflect.DeepEqual(gotTags, tt.wantTags) {
				t.Errorf(
					"parseFrontMatter() gotTags = %v, want %v",
					gotTags,
					tt.wantTags,
				)
			}
		})
	}
}

func TestFuzzyFinder_handleFuzzySelectError(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	type args struct {
		err error
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
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			f.handleFuzzySelectError(tt.args.err)
		})
	}
}

func TestFuzzyFinder_Execute(t *testing.T) {
	type fields struct {
		vaultDir string
		files    []string
	}
	type args struct {
		idx int
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
			f := &FuzzyFinder{
				vaultDir: tt.fields.vaultDir,
				files:    tt.fields.files,
			}
			f.Execute(tt.args.idx)
		})
	}
}
