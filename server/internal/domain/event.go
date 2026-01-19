// Package domain provides core domain models and business logic for the calendar application.
// This layer defines entities, validation rules, and data access contracts (repositories)
// independent of external frameworks or persistence mechanisms.
package domain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

// Event represents a calendar event (iCalendar VEVENT component)
// Hybrid storage: full ICS for CalDAV fidelity + extracted metadata for SQL queries
type Event struct {
	ID         int64
	CalendarID int64
	UID        string // iCalendar UID (globally unique)
	ICS        string // Full VEVENT component (stored as-is for CalDAV roundtrip)
	// Extracted metadata for efficient queries:
	Summary        string
	Description    string
	Location       string
	StartTime      time.Time
	EndTime        time.Time
	AllDay         bool
	RecurrenceRule string // RRULE string if recurring
	ETag           string // SHA-256 hash of ICS for conflict detection
	Sequence       int    // iCalendar SEQUENCE for versioning
	Status         string // TENTATIVE, CONFIRMED, CANCELLED
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// GenerateETag computes SHA-256 hash of ICS data for conflict detection
// Returns quoted string per HTTP spec: "abc123..."
func GenerateETag(icsData []byte) string {
	hash := sha256.Sum256(icsData)
	return fmt.Sprintf(`"%x"`, hash)
}

// Validate checks that required fields are present and valid
func (e *Event) Validate() error {
	if e.UID == "" {
		return errors.New("event UID is required")
	}
	if e.StartTime.IsZero() {
		return errors.New("event StartTime is required")
	}
	if !e.EndTime.IsZero() && e.EndTime.Before(e.StartTime) {
		return errors.New("event EndTime must be after or equal to StartTime")
	}
	if e.ICS == "" {
		return errors.New("event ICS data is required")
	}
	return nil
}
