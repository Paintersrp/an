package tui

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialPrompt(t *testing.T) {
	type args struct {
		cfgPath string
	}
	tests := []struct {
		name string
		args args
		want InitPromptModel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitialPrompt(tt.args.cfgPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitialPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitPromptModel_Init(t *testing.T) {
	type fields struct {
		focusIndex int
		inputs     []textinput.Model
		cursorMode cursor.Mode
		configPath string
	}
	tests := []struct {
		name   string
		fields fields
		want   tea.Cmd
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitPromptModel{
				focusIndex: tt.fields.focusIndex,
				inputs:     tt.fields.inputs,
				cursorMode: tt.fields.cursorMode,
				configPath: tt.fields.configPath,
			}
			if got := m.Init(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitPromptModel.Init() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitPromptModel_Update(t *testing.T) {
	type fields struct {
		focusIndex int
		inputs     []textinput.Model
		cursorMode cursor.Mode
		configPath string
	}
	type args struct {
		msg tea.Msg
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   tea.Model
		want1  tea.Cmd
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitPromptModel{
				focusIndex: tt.fields.focusIndex,
				inputs:     tt.fields.inputs,
				cursorMode: tt.fields.cursorMode,
				configPath: tt.fields.configPath,
			}
			got, got1 := m.Update(tt.args.msg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitPromptModel.Update() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("InitPromptModel.Update() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestInitPromptModel_updateInputs(t *testing.T) {
	type fields struct {
		focusIndex int
		inputs     []textinput.Model
		cursorMode cursor.Mode
		configPath string
	}
	type args struct {
		msg tea.Msg
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   tea.Cmd
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &InitPromptModel{
				focusIndex: tt.fields.focusIndex,
				inputs:     tt.fields.inputs,
				cursorMode: tt.fields.cursorMode,
				configPath: tt.fields.configPath,
			}
			if got := m.updateInputs(tt.args.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitPromptModel.updateInputs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitPromptModel_View(t *testing.T) {
	type fields struct {
		focusIndex int
		inputs     []textinput.Model
		cursorMode cursor.Mode
		configPath string
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
			m := InitPromptModel{
				focusIndex: tt.fields.focusIndex,
				inputs:     tt.fields.inputs,
				cursorMode: tt.fields.cursorMode,
				configPath: tt.fields.configPath,
			}
			if got := m.View(); got != tt.want {
				t.Errorf("InitPromptModel.View() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupDefaults(t *testing.T) {
	type args struct {
		homePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetupDefaults(tt.args.homePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetupDefaults() = %v, want %v", got, tt.want)
			}
		})
	}
}
