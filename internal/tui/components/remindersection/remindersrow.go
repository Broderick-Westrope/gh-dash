package remindersection

import (
	"time"

	"github.com/dlvhdr/gh-dash/v4/internal/utils"
)

func formatDueTime(remindAt time.Time) string {
	now := time.Now()
	if remindAt.Before(now) {
		elapsed := utils.TimeElapsed(remindAt)
		return elapsed + " ago"
	}
	remaining := time.Until(remindAt)
	return "in " + remaining.Round(time.Minute).String()
}

func truncateNote(note string, maxLen int) string {
	if len(note) <= maxLen {
		return note
	}
	return note[:maxLen-3] + "..."
}
