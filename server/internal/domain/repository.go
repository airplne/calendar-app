package domain

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrNotFound           = errors.New("not found")
	ErrPreconditionFailed = errors.New("precondition failed: ETag mismatch")
	ErrConflict           = errors.New("conflict: resource already exists")
)

// EventRepo defines the data access contract for events
type EventRepo interface {
	Create(ctx context.Context, event *Event) error
	GetByUID(ctx context.Context, calendarID int64, uid string) (*Event, error)
	GetByID(ctx context.Context, id int64) (*Event, error)
	List(ctx context.Context, calendarID int64, start, end time.Time) ([]*Event, error)
	ListAll(ctx context.Context, calendarID int64) ([]*Event, error)
	Update(ctx context.Context, event *Event, expectedETag string) error // Returns ErrPreconditionFailed if ETag mismatch
	Delete(ctx context.Context, calendarID int64, uid string) error
}

// CalendarRepo defines the data access contract for calendars
type CalendarRepo interface {
	Create(ctx context.Context, calendar *Calendar) error
	GetByID(ctx context.Context, id int64) (*Calendar, error)
	GetByName(ctx context.Context, userID int64, name string) (*Calendar, error)
	ListByUser(ctx context.Context, userID int64) ([]*Calendar, error)
	Update(ctx context.Context, calendar *Calendar) error
	Delete(ctx context.Context, id int64) error
	IncrementSyncToken(ctx context.Context, calendarID int64) (string, error)
}

// TaskRepo defines the data access contract for tasks
type TaskRepo interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	ListByUser(ctx context.Context, userID int64) ([]*Task, error)
	ListPending(ctx context.Context, userID int64) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id int64) error
}

// UserRepo defines the data access contract for users
type UserRepo interface {
	Create(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// User represents an authenticated user
type User struct {
	ID        int64
	Username  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
