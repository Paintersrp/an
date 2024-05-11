package submodels

import (
	tea "github.com/charmbracelet/bubbletea"
)

type SubmitButton struct {
	focused bool
}

func NewSubmitButton() SubmitButton {
	return SubmitButton{}
}

func (b *SubmitButton) Focus() {
	b.focused = true
}

func (b *SubmitButton) Blur() {
	b.focused = false
}

func (b SubmitButton) Update(msg tea.Msg) (SubmitButton, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if b.focused && msg.Type == tea.KeyEnter {
			return b, tea.Quit
		}
	}
	return b, nil
}

func (b SubmitButton) View() string {
	return "[ Submit ]"
}
