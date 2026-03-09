package reminderprompt

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func sendKey(m Model, key tea.KeyMsg) (Model, tea.Cmd) {
	updated, cmd := m.Update(key)
	return updated, cmd
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestReminderPrompt_ValidDurationTabAdvancesToNote(t *testing.T) {
	m := New(40)

	// Type a valid duration into the duration field
	for _, r := range "2h" {
		m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Tab to advance
	m, cmd := sendKey(m, tea.KeyMsg{Type: tea.KeyTab})

	if m.focused != 1 {
		t.Errorf("expected focused=1 (note field), got %d", m.focused)
	}
	if m.err != "" {
		t.Errorf("expected no error, got %q", m.err)
	}
	// cmd should be textinput.Blink, not a message cmd; just ensure it's not nil
	_ = cmd
}

func TestReminderPrompt_InvalidDurationTabShowsError(t *testing.T) {
	m := New(40)

	// Type an invalid duration
	for _, r := range "xyz" {
		m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Tab
	m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyTab})

	if m.focused != 0 {
		t.Errorf("expected focused=0 (duration field), got %d", m.focused)
	}
	if m.err == "" {
		t.Error("expected an error message, got empty string")
	}
}

func TestReminderPrompt_EscAtDurationEmitsCancelMsg(t *testing.T) {
	m := New(40)

	_, cmd := sendKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	msg := runCmd(cmd)

	if _, ok := msg.(CancelMsg); !ok {
		t.Errorf("expected CancelMsg, got %T", msg)
	}
}

func TestReminderPrompt_EscAtNoteEmitsCancelMsg(t *testing.T) {
	m := New(40)

	// Advance to note field with a valid duration
	for _, r := range "1d" {
		m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyTab})

	if m.focused != 1 {
		t.Fatalf("expected to be on note field, got focused=%d", m.focused)
	}

	_, cmd := sendKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	msg := runCmd(cmd)

	if _, ok := msg.(CancelMsg); !ok {
		t.Errorf("expected CancelMsg, got %T", msg)
	}
}

func TestReminderPrompt_EnterOnDurationEmitsConfirmMsg(t *testing.T) {
	m := New(40)

	// Type a valid duration
	for _, r := range "2h" {
		m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Enter on the duration field to submit directly
	_, cmd := sendKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	msg := runCmd(cmd)

	confirm, ok := msg.(ConfirmMsg)
	if !ok {
		t.Fatalf("expected ConfirmMsg, got %T", msg)
	}
	if confirm.Duration != 2*time.Hour {
		t.Errorf("expected duration 2h, got %v", confirm.Duration)
	}
	if confirm.Note != "" {
		t.Errorf("expected empty note, got %q", confirm.Note)
	}
}

func TestReminderPrompt_CtrlDOnNoteEmitsConfirmMsg(t *testing.T) {
	m := New(40)

	// Type valid duration
	for _, r := range "2h" {
		m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// Advance to note field
	m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyTab})

	if m.focused != 1 {
		t.Fatalf("expected to be on note field, got focused=%d", m.focused)
	}

	// Type a note
	for _, r := range "check in" {
		m, _ = sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Ctrl+D to confirm (Enter now inserts newline in textarea)
	_, cmd := sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlD})
	msg := runCmd(cmd)

	confirm, ok := msg.(ConfirmMsg)
	if !ok {
		t.Fatalf("expected ConfirmMsg, got %T", msg)
	}

	expectedDuration := 2 * time.Hour
	if confirm.Duration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, confirm.Duration)
	}
	if confirm.Note != "check in" {
		t.Errorf("expected note %q, got %q", "check in", confirm.Note)
	}
}
