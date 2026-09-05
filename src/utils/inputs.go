package utils

import (
	"log/slog"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TextInput(header string, footer string, placeholder string) (string, error) {
	slog.Debug("Getting text input", "header", header, "footer", footer, "placeholder", placeholder)
	m := initialModel(header, footer, placeholder)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		slog.Warn("Error while running bubbletea", "error", err)
		return "", err
	}

	slog.Debug("Got input", "value", m.textInput.Value())
	return m.textInput.Value(), nil
}

type (
	errMsg error
)

type model struct {
	textInput textinput.Model
	header    string
	footer    string
	err       error
	quitting  bool
}

func initialModel(header string, footer string, placeholder string) model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetVirtualCursor(false)
	ti.Focus()
	ti.CharLimit = 156
	ti.SetWidth(20)

	return model{textInput: ti, header: header, footer: footer}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	var c *tea.Cursor
	if !m.textInput.VirtualCursor() {
		c = m.textInput.Cursor()
		c.Y += lipgloss.Height(m.header)
	}

	str := lipgloss.JoinVertical(lipgloss.Top, m.header, m.textInput.View(), m.footer)
	if m.quitting {
		str += "\n"
	}

	v := tea.NewView(str)
	v.Cursor = c
	return v
}
