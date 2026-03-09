package prview

import (
	"strings"
	"testing"
	"time"

	"github.com/dlvhdr/gh-dash/v4/internal/data"
)

func TestReminderBanner(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		activeReminder *data.ReminderEntry
		wantEmpty      bool
		wantContains   []string
		wantAbsent     []string
	}{
		{
			name:           "no reminder returns empty string",
			activeReminder: nil,
			wantEmpty:      true,
		},
		{
			name: "pending reminder with note contains 'in' and the note",
			activeReminder: &data.ReminderEntry{
				RemindAt: now.Add(2 * time.Hour),
				Note:     "check the review comments",
			},
			wantContains: []string{"in", "check the review comments"},
		},
		{
			name: "pending reminder without note contains 'in' but no separator",
			activeReminder: &data.ReminderEntry{
				RemindAt: now.Add(30 * time.Minute),
				Note:     "",
			},
			wantContains: []string{"in"},
			wantAbsent:   []string{" — "},
		},
		{
			name: "due reminder contains 'ago'",
			activeReminder: &data.ReminderEntry{
				RemindAt: now.Add(-3 * time.Hour),
				Note:     "",
			},
			wantContains: []string{"ago"},
		},
		{
			name: "due reminder with note contains 'ago' and the note",
			activeReminder: &data.ReminderEntry{
				RemindAt: now.Add(-1 * time.Hour),
				Note:     "urgent review",
			},
			wantContains: []string{"ago", "urgent review"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Model{
				activeReminder: tc.activeReminder,
			}

			got := m.reminderBanner()

			if tc.wantEmpty {
				if got != "" {
					t.Errorf("expected empty string, got %q", got)
				}
				return
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("expected banner to contain %q, got %q", want, got)
				}
			}

			for _, absent := range tc.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("expected banner NOT to contain %q, got %q", absent, got)
				}
			}
		})
	}
}
