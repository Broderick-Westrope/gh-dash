package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempRemindersStore(t *testing.T) *RemindersStore {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "reminders.json")
	s := newRemindersStoreWithPath(filePath)
	// Flush synchronously before TempDir cleanup removes the directory.
	// t.Cleanup runs in LIFO order; registering here (after t.TempDir) means
	// this flush runs before the directory removal.
	t.Cleanup(func() { _ = s.Flush() })
	return s
}

func TestRemindersStore_SetAndGet(t *testing.T) {
	s := tempRemindersStore(t)
	key := ReminderKey("owner/repo", 42)
	entry := ReminderEntry{
		RemindAt: time.Now().Add(1 * time.Hour),
		Note:     "follow up",
	}

	s.Set(key, entry)

	got, ok := s.Get(key)
	if !ok {
		t.Fatalf("expected to find entry for key %q", key)
	}
	if !got.RemindAt.Equal(entry.RemindAt) {
		t.Errorf("RemindAt mismatch: got %v, want %v", got.RemindAt, entry.RemindAt)
	}
	if got.Note != entry.Note {
		t.Errorf("Note mismatch: got %q, want %q", got.Note, entry.Note)
	}
}

func TestRemindersStore_SetReplacesExisting(t *testing.T) {
	s := tempRemindersStore(t)
	key := ReminderKey("owner/repo", 1)

	s.Set(key, ReminderEntry{RemindAt: time.Now().Add(1 * time.Hour), Note: "original"})
	s.Set(key, ReminderEntry{RemindAt: time.Now().Add(2 * time.Hour), Note: "updated"})

	got, ok := s.Get(key)
	if !ok {
		t.Fatalf("expected to find entry for key %q", key)
	}
	if got.Note != "updated" {
		t.Errorf("expected updated note, got %q", got.Note)
	}
}

func TestRemindersStore_Remove(t *testing.T) {
	s := tempRemindersStore(t)
	key := ReminderKey("owner/repo", 7)

	s.Set(key, ReminderEntry{RemindAt: time.Now().Add(1 * time.Hour), Note: "to remove"})
	// Flush synchronously so the async goroutine doesn't race with TempDir cleanup.
	if err := s.Flush(); err != nil {
		t.Fatalf("flush after Set failed: %v", err)
	}
	s.Remove(key)
	if err := s.Flush(); err != nil {
		t.Fatalf("flush after Remove failed: %v", err)
	}

	_, ok := s.Get(key)
	if ok {
		t.Errorf("expected entry to be removed for key %q", key)
	}
}

func TestRemindersStore_RemoveNoOpsOnMissing(t *testing.T) {
	s := tempRemindersStore(t)
	// Should not panic or error
	s.Remove("nonexistent#99")
}

func TestRemindersStore_GetDueReturnsDueEntries(t *testing.T) {
	s := tempRemindersStore(t)

	pastKey := ReminderKey("owner/repo", 1)
	futureKey := ReminderKey("owner/repo", 2)

	s.Set(pastKey, ReminderEntry{RemindAt: time.Now().Add(-1 * time.Minute), Note: "past"})
	s.Set(futureKey, ReminderEntry{RemindAt: time.Now().Add(1 * time.Hour), Note: "future"})

	due := s.GetDue()

	if _, ok := due[pastKey]; !ok {
		t.Errorf("expected past entry to be in GetDue results")
	}
	if _, ok := due[futureKey]; ok {
		t.Errorf("expected future entry NOT to be in GetDue results")
	}
}

func TestRemindersStore_GetDueExcludesFutureEntries(t *testing.T) {
	s := tempRemindersStore(t)

	key := ReminderKey("owner/repo", 3)
	s.Set(key, ReminderEntry{RemindAt: time.Now().Add(24 * time.Hour), Note: "future"})

	due := s.GetDue()
	if len(due) != 0 {
		t.Errorf("expected no due entries, got %d", len(due))
	}
}

func TestRemindersStore_IsDue(t *testing.T) {
	s := tempRemindersStore(t)

	pastKey := ReminderKey("owner/repo", 10)
	futureKey := ReminderKey("owner/repo", 11)
	missingKey := ReminderKey("owner/repo", 12)

	s.Set(pastKey, ReminderEntry{RemindAt: time.Now().Add(-5 * time.Minute)})
	s.Set(futureKey, ReminderEntry{RemindAt: time.Now().Add(5 * time.Minute)})

	if !s.IsDue(pastKey) {
		t.Errorf("expected pastKey to be due")
	}
	if s.IsDue(futureKey) {
		t.Errorf("expected futureKey NOT to be due")
	}
	if s.IsDue(missingKey) {
		t.Errorf("expected missing key NOT to be due")
	}
}

func TestReminderKey_Format(t *testing.T) {
	key := ReminderKey("owner/repo", 42)
	expected := "owner/repo#42"
	if key != expected {
		t.Errorf("ReminderKey = %q, want %q", key, expected)
	}
}

func TestRemindersStore_PruneRemovesOldEntries(t *testing.T) {
	s := tempRemindersStore(t)

	// Entry older than 90 days — should be pruned
	oldKey := ReminderKey("owner/repo", 100)
	s.mu.Lock()
	s.entries[oldKey] = ReminderEntry{
		RemindAt: time.Now().Add(-91 * 24 * time.Hour),
		Note:     "old",
	}
	s.mu.Unlock()

	s.prune()

	_, ok := s.Get(oldKey)
	if ok {
		t.Errorf("expected old entry (>90 days) to be pruned")
	}
}

func TestRemindersStore_PruneRemovesZeroTimeEntries(t *testing.T) {
	s := tempRemindersStore(t)

	zeroKey := ReminderKey("owner/repo", 200)
	s.mu.Lock()
	s.entries[zeroKey] = ReminderEntry{
		RemindAt: time.Time{},
		Note:     "zero time",
	}
	s.mu.Unlock()

	s.prune()

	_, ok := s.Get(zeroKey)
	if ok {
		t.Errorf("expected zero-time entry to be pruned")
	}
}

func TestRemindersStore_PruneKeepsRecentEntries(t *testing.T) {
	s := tempRemindersStore(t)

	recentKey := ReminderKey("owner/repo", 300)
	s.mu.Lock()
	s.entries[recentKey] = ReminderEntry{
		RemindAt: time.Now().Add(-1 * 24 * time.Hour),
		Note:     "recent",
	}
	s.mu.Unlock()

	s.prune()

	_, ok := s.Get(recentKey)
	if !ok {
		t.Errorf("expected recent entry to be kept after prune")
	}
}

func TestRemindersStore_PersistsToDisk(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "reminders.json")

	s1 := newRemindersStoreWithPath(filePath)
	key := ReminderKey("owner/repo", 55)
	entry := ReminderEntry{RemindAt: time.Now().Add(1 * time.Hour).Truncate(time.Second), Note: "persisted"}
	s1.Set(key, entry)

	// Wait for async save to complete
	if err := s1.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file was written
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	// Load into a fresh store
	s2 := newRemindersStoreWithPath(filePath)
	got, ok := s2.Get(key)
	if !ok {
		t.Fatalf("expected entry to persist across store instances")
	}
	if got.Note != entry.Note {
		t.Errorf("Note mismatch after reload: got %q, want %q", got.Note, entry.Note)
	}
}
