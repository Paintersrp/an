package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Paintersrp/an/internal/state"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func Login(s *state.State) error {
	if _, err := tea.NewProgram(initialLoginModel(s)).Run(); err != nil {
		return err
	}

	return nil
}

type LoginModel struct {
	focusIndex int
	inputs     []textinput.Model
	state      *state.State
}

func initialLoginModel(s *state.State) LoginModel {
	m := LoginModel{
		inputs: make([]textinput.Model, 2),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch i {
		case 0:
			t.Placeholder = "Email"
			t.CharLimit = 64
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}

		m.inputs[i] = t
	}

	m.state = s

	return m
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" && m.focusIndex == len(m.inputs) {
				// TODO: Validate?

				loginData := map[string]string{
					"email":    m.inputs[0].Value(),
					"password": m.inputs[1].Value(),
				}

				loginDataJson, err := json.Marshal(loginData)
				if err != nil {
					fmt.Printf("failed to encode login data to JSON: %v", err)
					return m, tea.Quit
				}

				resp, err := http.Post(
					"http://localhost:6474/v1/auth/login",
					"application/json",
					bytes.NewBuffer(loginDataJson),
				)
				if err != nil {
					fmt.Printf("failed to log in: %v", err)
					return m, tea.Quit
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					fmt.Println("status code: ", resp.StatusCode)
					return m, tea.Quit
				}

				var respData map[string]string
				err = json.NewDecoder(resp.Body).Decode(&respData)
				if err != nil {
					fmt.Printf("failed to decode response: %v", err)
					return m, tea.Quit
				}

				token, ok := respData["token"]
				if !ok {
					fmt.Printf("token not found in response: %v", err)
					return m, tea.Quit
				}

				m.state.Config.ChangeToken(token)
				fmt.Println("Login successful!")

				return m, tea.Quit

			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *LoginModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m LoginModel) View() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	return b.String()
}
