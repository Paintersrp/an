// Package templater provides functionality to manage and render templates for Zettelkasten notes.
package templater

import (
	"reflect"
	"testing"
)

func TestNewTemplater(t *testing.T) {
	tests := []struct {
		name    string
		want    *Templater
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplater()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTemplater() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplater() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplater_Execute(t *testing.T) {
	type fields struct {
		templates TemplateMap
	}
	type args struct {
		templateName string
		data         interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Templater{
				templates: tt.fields.templates,
			}
			got, err := tr.Execute(tt.args.templateName, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Templater.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Templater.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplater_GenerateTagsAndDate(t *testing.T) {
	type fields struct {
		templates TemplateMap
	}
	type args struct {
		tmplName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
		want1  []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Templater{
				templates: tt.fields.templates,
			}
			got, got1 := tr.GenerateTagsAndDate(tt.args.tmplName)
			if got != tt.want {
				t.Errorf("Templater.GenerateTagsAndDate() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Templater.GenerateTagsAndDate() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
