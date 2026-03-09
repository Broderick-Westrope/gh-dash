package reminderprompt

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dlvhdr/gh-dash/v4/internal/data"
)

// ConfirmMsg is emitted when the user confirms the reminder.
type ConfirmMsg struct {
	Duration time.Duration
	Note     string
}

// CancelMsg is emitted when the user cancels.
type CancelMsg struct{}

type Model struct {
	durationInput textinput.Model
	noteInput     textinput.Model
	focused       int // 0 = duration, 1 = note
	err           string
	width         int
}

func New(width int) Model {
	dur := textinput.New()
	dur.Placeholder = "e.g. 2h, 30m, 1d"
	dur.Focus()
	dur.CharLimit = 20

	note := textinput.New()
	note.Placeholder = "optional note"
	note.CharLimit = 100

	m := Model{
		durationInput: dur,
		noteInput:     note,
		focused:       0,
		width:         width,
	}
	m.SetWidth(width)
	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyEsc || msg.String() == "ctrl+c":
			return m, func() tea.Msg { return CancelMsg{} }

		case (msg.Type == tea.KeyTab || msg.Type == tea.KeyEnter) && m.focused == 0:
			dur, err := data.ParseDuration(m.durationInput.Value())
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.err = ""
			_ = dur
			m.focused = 1
			m.durationInput.Blur()
			m.noteInput.Focus()
			return m, textinput.Blink

		case msg.Type == tea.KeyEnter && m.focused == 1:
			dur, err := data.ParseDuration(m.durationInput.Value())
			if err != nil {
				// Shouldn't happen, but handle gracefully
				m.err = err.Error()
				m.focused = 0
				m.noteInput.Blur()
				m.durationInput.Focus()
				return m, nil
			}
			return m, func() tea.Msg {
				return ConfirmMsg{Duration: dur, Note: m.noteInput.Value()}
			}

		case msg.Type == tea.KeyShiftTab:
			m.focused = 0
			m.noteInput.Blur()
			m.durationInput.Focus()
			return m, textinput.Blink

		default:
			if m.focused == 0 {
				m.durationInput, cmd = m.durationInput.Update(msg)
			} else {
				m.noteInput, cmd = m.noteInput.Update(msg)
			}
			return m, cmd
		}
	}

	// Route non-key messages to both inputs
	if m.focused == 0 {
		m.durationInput, cmd = m.durationInput.Update(msg)
	} else {
		m.noteInput, cmd = m.noteInput.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	errLine := ""
	if m.err != "" {
		errLine = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.err)
	}

	durationField := lipgloss.JoinVertical(lipgloss.Left,
		"Duration:",
		m.durationInput.View()+errLine,
	)

	noteField := lipgloss.JoinVertical(lipgloss.Left,
		"Note:",
		m.noteInput.View(),
	)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		durationField,
		"",
		noteField,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(m.width - 2).
		Render(inner)
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.durationInput.Width = w - 6
	m.noteInput.Width = w - 6
}
