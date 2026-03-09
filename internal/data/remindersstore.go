package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// ReminderEntry holds the details for a single PR reminder.
type ReminderEntry struct {
	RemindAt time.Time `json:"remindAt"`
	Note     string    `json:"note"`
}

// RemindersStore persists PR reminders to disk.
type RemindersStore struct {
	mu       sync.RWMutex
	entries  map[string]ReminderEntry
	filePath string
	saveWg   sync.WaitGroup // tracks in-flight async saves
}

// ReminderKey returns the canonical key for a PR reminder.
// Format: "owner/repo#42"
func ReminderKey(repoNameWithOwner string, number int) string {
	return fmt.Sprintf("%s#%d", repoNameWithOwner, number)
}

func newRemindersStore(filename string) *RemindersStore {
	store := &RemindersStore{
		entries: make(map[string]ReminderEntry),
	}
	filePath, err := getStateFilePath(filename)
	if err != nil {
		log.Error("Failed to get state file path for reminders", "err", err)
	}
	store.filePath = filePath
	if err := store.load(); err != nil {
		log.Error("Failed to load reminders", "err", err)
	}
	return store
}

// newRemindersStoreWithPath creates a RemindersStore with an explicit file path.
// Intended for use in tests to avoid requiring the XDG state directory.
func newRemindersStoreWithPath(filePath string) *RemindersStore {
	store := &RemindersStore{
		entries:  make(map[string]ReminderEntry),
		filePath: filePath,
	}
	if err := store.load(); err != nil {
		log.Error("Failed to load reminders", "err", err)
	}
	return store
}

// Set adds or replaces a reminder entry for the given key.
func (s *RemindersStore) Set(key string, entry ReminderEntry) {
	s.mu.Lock()
	s.entries[key] = entry
	s.mu.Unlock()
	s.saveWg.Add(1)
	go func() {
		defer s.saveWg.Done()
		s.save() //nolint:errcheck
	}()
}

// Remove deletes the reminder for the given key. No-op if absent.
func (s *RemindersStore) Remove(key string) {
	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
	s.saveWg.Add(1)
	go func() {
		defer s.saveWg.Done()
		s.save() //nolint:errcheck
	}()
}

// Get returns the reminder entry for the given key.
func (s *RemindersStore) Get(key string) (ReminderEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[key]
	return entry, ok
}

// GetDue returns all entries whose RemindAt is before time.Now().
func (s *RemindersStore) GetDue() map[string]ReminderEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	due := make(map[string]ReminderEntry)
	for k, e := range s.entries {
		if e.RemindAt.Before(now) {
			due[k] = e
		}
	}
	return due
}

// GetAll returns a copy of all reminder entries.
func (s *RemindersStore) GetAll() map[string]ReminderEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]ReminderEntry, len(s.entries))
	for k, e := range s.entries {
		result[k] = e
	}
	return result
}

// IsDue returns true if the reminder for the given key is due.
func (s *RemindersStore) IsDue(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[key]
	if !ok {
		return false
	}
	return entry.RemindAt.Before(time.Now())
}

func (s *RemindersStore) save() error {
	if s.filePath == "" {
		return nil
	}

	// Snapshot entries under a short read lock, then do I/O without holding it.
	s.mu.RLock()
	snapshot := make(map[string]ReminderEntry, len(s.entries))
	for k, v := range s.entries {
		snapshot[k] = v
	}
	s.mu.RUnlock()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	log.Debug("Saved reminders", "count", len(snapshot))
	return nil
}

func (s *RemindersStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var entries map[string]ReminderEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	s.entries = entries
	s.prune()
	log.Debug("Loaded reminders", "count", len(s.entries))
	return nil
}

// prune removes entries where RemindAt is zero or more than 90 days in the past.
func (s *RemindersStore) prune() {
	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	for key, entry := range s.entries {
		if entry.RemindAt.IsZero() || entry.RemindAt.Before(cutoff) {
			delete(s.entries, key)
		}
	}
}

// Flush waits for all in-flight async saves to complete, then performs one
// final synchronous save. This ensures data is on disk before the caller
// proceeds (useful in tests and on graceful shutdown).
func (s *RemindersStore) Flush() error {
	s.saveWg.Wait()
	return s.save()
}

// Singleton

var (
	remindersStore     *RemindersStore
	remindersStoreOnce sync.Once
)

// GetRemindersStore returns the singleton reminders store.
func GetRemindersStore() *RemindersStore {
	remindersStoreOnce.Do(func() {
		remindersStore = newRemindersStore("reminders.json")
	})
	return remindersStore
}
