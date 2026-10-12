// Package cmd tests task search date validation.
package cmd

import "testing"

func TestParseDateOnlyAcceptsCalendarDates(t *testing.T) {
	for _, input := range []string{"2024-12-31", "2024-02-29", "2025-01-01"} {
		got, err := parseDateOnly(input)
		if err != nil {
			t.Fatalf("parseDateOnly(%q) error = %v", input, err)
		}
		if got != input {
			t.Fatalf("parseDateOnly(%q) = %q", input, got)
		}
	}
}

func TestParseDateOnlyRejectsInvalidDates(t *testing.T) {
	for _, input := range []string{"2024-13-45", "2025-02-29", "xxxx-xx-xx", "2024-1-05", "2024-12-31T10:00:00", ""} {
		if _, err := parseDateOnly(input); err == nil {
			t.Fatalf("parseDateOnly(%q) accepted an invalid date", input)
		}
	}
}
