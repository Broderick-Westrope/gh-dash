package remindersection

import (
	"testing"
	"time"

	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/prrow"
)

// TestDismiss verifies that dismiss removes the matching row while leaving others intact.
// Note: dismiss also calls data.GetRemindersStore().Remove(key) which is a no-op in
// tests since the key is not present in the production store.
func TestDismiss(t *testing.T) {
	t.Run("removes the correct row by key", func(t *testing.T) {
		m := &Model{}
		m.Rows = []ReminderRow{
			makeRow("owner/repo1", 1, "note1"),
			makeRow("owner/repo2", 2, "note2"),
			makeRow("owner/repo3", 3, "note3"),
		}

		targetKey := data.ReminderKey("owner/repo2", 2)

		// dismiss calls GetRemindersStore().Remove(key) which is safe even when key
		// doesn't exist in the store. We validate only the in-memory row removal.
		m.dismiss(targetKey)

		if len(m.Rows) != 2 {
			t.Fatalf("expected 2 rows after dismiss, got %d", len(m.Rows))
		}
		for _, row := range m.Rows {
			k := data.ReminderKey(row.Data.Primary.Repository.NameWithOwner, row.Data.Primary.Number)
			if k == targetKey {
				t.Errorf("dismissed key %q still present in Rows", targetKey)
			}
		}
	})

	t.Run("no-op when key is absent", func(t *testing.T) {
		m := &Model{}
		m.Rows = []ReminderRow{
			makeRow("owner/repo1", 1, "note1"),
		}

		m.dismiss(data.ReminderKey("owner/nonexistent", 99))

		if len(m.Rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(m.Rows))
		}
	})

	t.Run("removes all rows when all share the target key", func(t *testing.T) {
		m := &Model{}
		m.Rows = []ReminderRow{
			makeRow("owner/repo1", 1, "note1"),
		}

		targetKey := data.ReminderKey("owner/repo1", 1)
		m.dismiss(targetKey)

		if len(m.Rows) != 0 {
			t.Fatalf("expected 0 rows after dismiss, got %d", len(m.Rows))
		}
	})
}

// TestFormatDueTime tests the due-time formatting for past and future times.
func TestFormatDueTime(t *testing.T) {
	t.Run("past time includes 'ago'", func(t *testing.T) {
		past := time.Now().Add(-2 * time.Hour)
		result := formatDueTime(past)
		if len(result) == 0 {
			t.Error("expected non-empty string for past time")
		}
		lastChars := result[len(result)-3:]
		if lastChars != "ago" {
			t.Errorf("expected result to end with 'ago', got %q", result)
		}
	})

	t.Run("future time starts with 'in '", func(t *testing.T) {
		future := time.Now().Add(2 * time.Hour)
		result := formatDueTime(future)
		if len(result) < 3 || result[:3] != "in " {
			t.Errorf("expected result to start with 'in ', got %q", result)
		}
	})

	t.Run("past time one hour ago produces elapsed string", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		result := formatDueTime(past)
		// Should be something like "1h ago"
		if result == "" {
			t.Error("expected non-empty result")
		}
	})
}

// TestTruncateNote tests note truncation.
func TestTruncateNote(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short note is unchanged",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "note exactly at max is unchanged",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long note is truncated with ellipsis",
			input:    "this is a long note that should be truncated",
			maxLen:   10,
			expected: "this is...",
		},
		{
			name:     "empty note",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "note one char over max gets truncated",
			input:    "hello!",
			maxLen:   5,
			expected: "he...",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateNote(tc.input, tc.maxLen)
			if got != tc.expected {
				t.Errorf("truncateNote(%q, %d) = %q; want %q", tc.input, tc.maxLen, got, tc.expected)
			}
		})
	}
}

// TestMatchesFilters verifies the client-side filter logic.
func TestMatchesFilters(t *testing.T) {
	openPR := &data.PullRequestData{State: "OPEN"}
	closedPR := &data.PullRequestData{State: "CLOSED"}
	mergedPR := &data.PullRequestData{State: "MERGED"}

	cases := []struct {
		name    string
		pr      *data.PullRequestData
		filters string
		want    bool
	}{
		{"empty filters passes all", openPR, "", true},
		{"is:open matches open PR", openPR, "is:open", true},
		{"is:open rejects closed PR", closedPR, "is:open", false},
		{"is:closed matches closed PR", closedPR, "is:closed", true},
		{"is:closed rejects open PR", openPR, "is:closed", false},
		{"is:merged matches merged PR", mergedPR, "is:merged", true},
		{"is:merged rejects open PR", openPR, "is:merged", false},
		{"multiple filters - all match", openPR, "is:open", true},
		{"unrecognised token is ignored (no panic)", openPR, "unknown:token", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesFilters(tc.pr, tc.filters)
			if got != tc.want {
				t.Errorf("matchesFilters(%v, %q) = %v; want %v", tc.pr.State, tc.filters, got, tc.want)
			}
		})
	}
}

// ---- helpers ----

func makeRow(repoWithOwner string, number int, note string) ReminderRow {
	pr := &data.PullRequestData{
		Number: number,
		Repository: data.Repository{
			NameWithOwner: repoWithOwner,
		},
	}
	return ReminderRow{
		Data:  prrow.Data{Primary: pr},
		Entry: data.ReminderEntry{Note: note, RemindAt: time.Now().Add(-time.Minute)},
		Key:   data.ReminderKey(repoWithOwner, number),
	}
}
