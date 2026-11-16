package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// SlotLength is the length of a grid slot, and of the average of each of its values.
	SlotLength = 3 * time.Minute
	// SearchWindow is how far before a slot the server looks for a slot that GSE published.
	SearchWindow = 30 * time.Minute
	// Retention is how long GSE keeps a file.
	Retention = 31 * 24 * time.Hour
)

// SlotOf returns the start of the grid slot that holds t, in kernel.TbilisiZone().
func SlotOf(t time.Time) time.Time {
	local := t.In(kernel.TbilisiZone())
	step := int(SlotLength / time.Minute)

	return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(),
		local.Minute()/step*step, 0, 0, kernel.TbilisiZone())
}

// ParseMoment takes RFC 3339 with an offset, or a time of Tbilisi with no offset:
// "2026-11-16T13:01" or "2026-11-16 13:01", with or without seconds.
func ParseMoment(text string) (time.Time, error) {
	text = strings.TrimSpace(text)

	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04Z07:00"} {
		if moment, err := time.Parse(layout, text); err == nil {
			return moment, nil
		}
	}

	for _, layout := range []string{
		"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02 15:04",
	} {
		if moment, err := time.ParseInLocation(layout, text, kernel.TbilisiZone()); err == nil {
			return moment, nil
		}
	}

	return time.Time{}, fmt.Errorf("%q: %w", text, ErrInvalidMoment)
}
