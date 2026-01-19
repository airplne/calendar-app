package domain

import (
	"errors"
	"time"
)

// Calendar represents a CalDAV calendar collection
type Calendar struct {
	ID          int64
	UserID      int64
	Name        string // URL-safe identifier (e.g., "personal")
	DisplayName string // Human-readable name
	Color       string // Hex color code (e.g., "#FF5733")
	Description string
	SyncToken   string // Internal revision token for change tracking
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate checks required fields
func (c *Calendar) Validate() error {
	if c.Name == "" {
		return errors.New("calendar Name is required")
	}
	if c.UserID <= 0 {
		return errors.New("calendar UserID must be greater than 0")
	}
	return nil
}
