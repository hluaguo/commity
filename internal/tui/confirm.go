package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmModel is a custom component for the confirm step
// Shows: Yes - commit, Cancel, Regenerate with inline feedback input
type ConfirmModel struct {
	cursor    int // 0: commit, 1: cancel, 2: regenerate
	input     textinput.Model
	theme     *Theme
	submitted bool
	action    string // "commit", "cancel", "regenerate"
	feedback  string
}

func NewConfirmModel(theme *Theme) *ConfirmModel {
	ti := textinput.New()
	ti.Placeholder = "feedback..."
	ti.CharLimit = 200
	ti.Width = 30

	return &ConfirmModel{
		cursor: 0,
		input:  ti,
		theme:  theme,
	}
}

func (m *ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m *ConfirmModel) Update(msg tea.Msg) (*ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor != 2 {
					m.input.Blur()
				}
			}
			return m, nil

		case "down", "j":
			if m.cursor < 2 {
				m.cursor++
				if m.cursor == 2 {
					m.input.Focus()
					return m, textinput.Blink
				}
			}
			return m, nil

		case "enter":
			m.submitted = true
			switch m.cursor {
			case 0:
				m.action = "commit"
			case 1:
				m.action = "cancel"
			case 2:
				m.action = "regenerate"
				m.feedback = m.input.Value()
			}
			return m, nil

		case "e", "E":
			// Edit message - only when not typing in regenerate input
			if m.cursor != 2 || !m.input.Focused() {
				m.submitted = true
				m.action = "edit"
				return m, nil
			}

		default:
			// If on regenerate option, any printable key focuses and types
			if m.cursor == 2 {
				if !m.input.Focused() {
					m.input.Focus()
				}
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
	}

	// Update text input if focused
	if m.cursor == 2 {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *ConfirmModel) View() string {
	var s strings.Builder

	options := []string{"Yes - commit", "Cancel"}

	selectedStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(m.theme.Secondary)
	dimStyle := lipgloss.NewStyle().Foreground(m.theme.Dim)

	s.WriteString(dimStyle.Render("What do you want to do?"))
	s.WriteString("\n\n")

	for i, opt := range options {
		cursor := "  "
		style := normalStyle
		if m.cursor == i {
			cursor = "> "
			style = selectedStyle
		}
		s.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(opt)))
	}

	// Regenerate option with inline input
	cursor := "  "
	style := normalStyle
	if m.cursor == 2 {
		cursor = "> "
		style = selectedStyle
	}

	inputView := m.input.View()
	if !m.input.Focused() && m.input.Value() == "" {
		inputView = dimStyle.Render("type feedback...")
	}

	s.WriteString(fmt.Sprintf("%s%s %s", cursor, style.Render("Regenerate:"), inputView))
	s.WriteString("\n\n")

	// Key hint
	keyStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	s.WriteString(dimStyle.Render("Press "))
	s.WriteString(keyStyle.Render("e"))
	s.WriteString(dimStyle.Render(" to edit message"))

	return s.String()
}

func (m *ConfirmModel) Submitted() bool {
	return m.submitted
}

func (m *ConfirmModel) Action() string {
	return m.action
}

func (m *ConfirmModel) Feedback() string {
	return m.feedback
}
