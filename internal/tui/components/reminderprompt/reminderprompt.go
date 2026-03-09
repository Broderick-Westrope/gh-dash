package reminderprompt

import (
	"time"

	"github.com/charmbracelet/bubbles/textarea"
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
	noteInput     textarea.Model
	focused       int // 0 = duration, 1 = note
	err           string
	width         int
}

func New(width int) Model {
	dur := textinput.New()
	dur.Placeholder = "e.g. 2h, 30m, 1d"
	dur.Focus()
	dur.CharLimit = 20

	note := textarea.New()
	note.Placeholder = "optional note"
	note.SetHeight(3)
	note.SetWidth(width - 6)
	note.CharLimit = 500
	note.ShowLineNumbers = false

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

		case m.focused == 0:
			switch msg.Type {
			case tea.KeyTab:
				dur, err := data.ParseDuration(m.durationInput.Value())
				if err != nil {
					m.err = err.Error()
					return m, nil
				}
				m.err = ""
				// dur is validated but not stored; canonical parse happens at submit time.
				_ = dur
				m.durationInput.Blur()
				m.focused = 1
				cmd = m.noteInput.Focus()
				return m, cmd

			case tea.KeyEnter:
				dur, err := data.ParseDuration(m.durationInput.Value())
				if err != nil {
					m.err = err.Error()
					return m, nil
				}
				m.err = ""
				return m, func() tea.Msg {
					return ConfirmMsg{Duration: dur, Note: m.noteInput.Value()}
				}

			case tea.KeyShiftTab:
				// no-op: already on the first field
				return m, nil

			default:
				if msg.Type == tea.KeyCtrlD {
					dur, err := data.ParseDuration(m.durationInput.Value())
					if err != nil {
						m.err = err.Error()
						return m, nil
					}
					m.err = ""
					return m, func() tea.Msg {
						return ConfirmMsg{Duration: dur, Note: m.noteInput.Value()}
					}
				}
				m.durationInput, cmd = m.durationInput.Update(msg)
				return m, cmd
			}

		case m.focused == 1:
			switch msg.Type {
			// Tab on the note field is intentionally treated as back-navigation to duration.
			case tea.KeyTab, tea.KeyShiftTab:
				m.noteInput.Blur()
				m.focused = 0
				m.durationInput.Focus()
				return m, textinput.Blink

			default:
				if msg.Type == tea.KeyCtrlD {
					dur, err := data.ParseDuration(m.durationInput.Value())
					if err != nil {
						m.err = err.Error()
						m.noteInput.Blur()
						m.focused = 0
						m.durationInput.Focus()
						return m, textinput.Blink
					}
					m.err = ""
					return m, func() tea.Msg {
						return ConfirmMsg{Duration: dur, Note: m.noteInput.Value()}
					}
				}
				// Pass all other keys (including Enter) to the textarea
				m.noteInput, cmd = m.noteInput.Update(msg)
				return m, cmd
			}
		}
	}

	// Route non-key messages to the active input
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

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Tab: switch fields • Ctrl+D: submit • Esc: cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		durationField,
		"",
		noteField,
		"",
		hint,
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
	m.noteInput.SetWidth(w - 6)
}
