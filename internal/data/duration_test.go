package data

import (
	"testing"
	"time"
)

func TestParseDuration_ValidSingleUnits(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"30m", 30 * time.Minute},
		{"2h", 2 * time.Hour},
		{"1d", 24 * time.Hour},
		{"45s", 45 * time.Second},
		{"1s", 1 * time.Second},
		{"0m", 0},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseDuration(tc.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) returned unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseDuration_ValidMultiPart(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"2h 30m", 150 * time.Minute},
		{"1d 4h", 28 * time.Hour},
		{"1h30m", 90 * time.Minute},
		{"1d 4h 30m", 28*time.Hour + 30*time.Minute},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseDuration(tc.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) returned unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseDuration_Errors(t *testing.T) {
	tests := []struct {
		input string
	}{
		{""},
		{"abc"},
		{"2x"},
		{"2"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			_, err := ParseDuration(tc.input)
			if err == nil {
				t.Errorf("ParseDuration(%q) expected error, got nil", tc.input)
			}
		})
	}
}
