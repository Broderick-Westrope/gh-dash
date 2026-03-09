package data

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var durationTokenRe = regexp.MustCompile(`(\d+)([dhms])`)

// ParseDuration parses a human-friendly duration string into a time.Duration.
//
// Supported formats:
//   - Single unit:  "30m", "2h", "1d", "45s"
//   - Multi-part:   "2h 30m", "1d 4h", "1h30m", "1d 4h 30m"
//
// 1d is defined as 24 * time.Hour.
// Returns an error for unrecognised input.
func ParseDuration(s string) (time.Duration, error) {
	matches := durationTokenRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("unrecognised duration %q: use formats like 30m, 2h, 1d, 2h 30m", s)
	}

	// Verify the entire string is composed of valid tokens (with optional whitespace).
	// Remove all matched tokens from the input; only whitespace should remain.
	stripped := durationTokenRe.ReplaceAllString(s, "")
	for _, r := range stripped {
		if r != ' ' && r != '\t' {
			return 0, fmt.Errorf("unrecognised duration %q: use formats like 30m, 2h, 1d, 2h 30m", s)
		}
	}

	var total time.Duration
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, fmt.Errorf("unrecognised duration %q: use formats like 30m, 2h, 1d, 2h 30m", s)
		}
		unit := m[2]
		switch unit {
		case "d":
			total += time.Duration(n) * 24 * time.Hour
		case "h":
			total += time.Duration(n) * time.Hour
		case "m":
			total += time.Duration(n) * time.Minute
		case "s":
			total += time.Duration(n) * time.Second
		default:
			return 0, fmt.Errorf("unrecognised duration %q: use formats like 30m, 2h, 1d, 2h 30m", s)
		}
	}
	return total, nil
}
