package domain

import (
	"errors"
	"fmt"
	"time"
)

// Task represents a Todoist-synced or local task
type Task struct {
	ID          int64
	UserID      int64
	TodoistID   *string // Nullable - nil for local tasks
	Content     string
	Description string
	Priority    int // 1-4 (Todoist priority levels)
	DueDate     *time.Time
	Completed   bool
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate checks required fields
func (t *Task) Validate() error {
	if t.Content == "" {
		return errors.New("task Content is required")
	}
	if t.Priority < 1 || t.Priority > 4 {
		return fmt.Errorf("task Priority must be between 1 and 4, got %d", t.Priority)
	}
	return nil
}
