package caldav

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"

	"github.com/airplne/calendar-app/server/internal/data"
	"github.com/airplne/calendar-app/server/internal/domain"
)

// Backend implements caldav.Backend using our domain repositories
type Backend struct {
	db           *sql.DB
	userRepo     domain.UserRepo
	calendarRepo *data.SQLiteCalendarRepo
	eventRepo    *data.SQLiteEventRepo

	// Current authenticated user (set by auth middleware via context)
	// For MVP single-user, we'll use a fixed user
}

// NewBackend creates a new CalDAV backend
// The db parameter is required for transaction support (atomic event write + sync token bump).
// The calendarRepo and eventRepo must be concrete SQLite repos to support WithTx.
func NewBackend(db *sql.DB, userRepo domain.UserRepo, calendarRepo *data.SQLiteCalendarRepo, eventRepo *data.SQLiteEventRepo) *Backend {
	return &Backend{
		db:           db,
		userRepo:     userRepo,
		calendarRepo: calendarRepo,
		eventRepo:    eventRepo,
	}
}

// CalendarHomeSetPath returns the path to the user's calendar collection
// Format: /dav/calendars/{username}/
func (b *Backend) CalendarHomeSetPath(ctx context.Context) (string, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return "", fmt.Errorf("no authenticated user in context")
	}
	return fmt.Sprintf("/dav/calendars/%s/", user.Username), nil
}

// CurrentUserPrincipal returns the principal URL for the authenticated user
// Implements webdav.UserPrincipalBackend
func (b *Backend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return "", fmt.Errorf("no authenticated user in context")
	}
	return fmt.Sprintf("/dav/principals/%s/", user.Username), nil
}

// ListCalendars returns all calendars for the current user
func (b *Backend) ListCalendars(ctx context.Context) ([]caldav.Calendar, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("no authenticated user")
	}

	cals, err := b.calendarRepo.ListByUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	result := make([]caldav.Calendar, 0, len(cals))
	for _, cal := range cals {
		result = append(result, b.domainCalendarToCalDAV(cal, user.Username))
	}
	return result, nil
}

// GetCalendar retrieves a single calendar by path
func (b *Backend) GetCalendar(ctx context.Context, urlPath string) (*caldav.Calendar, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("no authenticated user")
	}

	// Parse calendar name from path: /dav/calendars/{username}/{calendar-name}/
	calName := extractCalendarName(urlPath)
	if calName == "" {
		return nil, webdav.NewHTTPError(404, fmt.Errorf("calendar not found"))
	}

	cal, err := b.calendarRepo.GetByName(ctx, user.ID, calName)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, webdav.NewHTTPError(404, fmt.Errorf("calendar not found"))
		}
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}

	result := b.domainCalendarToCalDAV(cal, user.Username)
	return &result, nil
}

// CreateCalendar creates a new calendar
func (b *Backend) CreateCalendar(ctx context.Context, calendar *caldav.Calendar) error {
	user := getUserFromContext(ctx)
	if user == nil {
		return fmt.Errorf("no authenticated user")
	}

	calName := extractCalendarName(calendar.Path)

	domainCal := &domain.Calendar{
		UserID:      user.ID,
		Name:        calName,
		DisplayName: calendar.Name,
		Description: calendar.Description,
	}

	if err := b.calendarRepo.Create(ctx, domainCal); err != nil {
		if err == domain.ErrConflict {
			return webdav.NewHTTPError(409, fmt.Errorf("calendar already exists"))
		}
		return fmt.Errorf("failed to create calendar: %w", err)
	}

	slog.Info("caldav.calendar.created", "username", user.Username, "calendar", calName)
	return nil
}

// GetCalendarObject retrieves a single event by path
func (b *Backend) GetCalendarObject(ctx context.Context, urlPath string, req *caldav.CalendarCompRequest) (*caldav.CalendarObject, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("no authenticated user")
	}

	// Parse: /dav/calendars/{username}/{calendar}/{uid}.ics
	calName, uid := extractCalendarAndUID(urlPath)
	if calName == "" || uid == "" {
		return nil, webdav.NewHTTPError(404, fmt.Errorf("event not found"))
	}

	cal, err := b.calendarRepo.GetByName(ctx, user.ID, calName)
	if err != nil {
		return nil, webdav.NewHTTPError(404, fmt.Errorf("calendar not found"))
	}

	event, err := b.eventRepo.GetByUID(ctx, cal.ID, uid)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, webdav.NewHTTPError(404, fmt.Errorf("event not found"))
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return b.domainEventToCalDAV(event, urlPath), nil
}

// ListCalendarObjects returns all events in a calendar
func (b *Backend) ListCalendarObjects(ctx context.Context, urlPath string, req *caldav.CalendarCompRequest) ([]caldav.CalendarObject, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("no authenticated user")
	}

	calName := extractCalendarName(urlPath)
	cal, err := b.calendarRepo.GetByName(ctx, user.ID, calName)
	if err != nil {
		return nil, webdav.NewHTTPError(404, fmt.Errorf("calendar not found"))
	}

	events, err := b.eventRepo.ListAll(ctx, cal.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	result := make([]caldav.CalendarObject, 0, len(events))
	for _, event := range events {
		objPath := fmt.Sprintf("%s%s.ics", ensureTrailingSlash(urlPath), event.UID)
		result = append(result, *b.domainEventToCalDAV(event, objPath))
	}
	return result, nil
}

// QueryCalendarObjects queries events with filters (time range, etc.)
func (b *Backend) QueryCalendarObjects(ctx context.Context, urlPath string, query *caldav.CalendarQuery) ([]caldav.CalendarObject, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("no authenticated user")
	}

	calName := extractCalendarName(urlPath)
	cal, err := b.calendarRepo.GetByName(ctx, user.ID, calName)
	if err != nil {
		return nil, webdav.NewHTTPError(404, fmt.Errorf("calendar not found"))
	}

	// For MVP: if query has time-range filter, use List; otherwise ListAll
	var events []*domain.Event
	if query != nil && !query.CompFilter.Start.IsZero() && !query.CompFilter.End.IsZero() {
		events, err = b.eventRepo.List(ctx, cal.ID, query.CompFilter.Start, query.CompFilter.End)
	} else {
		events, err = b.eventRepo.ListAll(ctx, cal.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	result := make([]caldav.CalendarObject, 0, len(events))
	for _, event := range events {
		objPath := fmt.Sprintf("%s%s.ics", ensureTrailingSlash(urlPath), event.UID)
		result = append(result, *b.domainEventToCalDAV(event, objPath))
	}
	return result, nil
}

// PutCalendarObject creates or updates an event
// This is the CRITICAL method for CalDAV sync with ETag conflict detection
// Event write and sync token bump are atomic (single transaction).
func (b *Backend) PutCalendarObject(ctx context.Context, urlPath string, icalData *ical.Calendar, opts *caldav.PutCalendarObjectOptions) (*caldav.CalendarObject, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("no authenticated user")
	}

	calName, uid := extractCalendarAndUID(urlPath)
	if calName == "" {
		return nil, webdav.NewHTTPError(400, fmt.Errorf("invalid path"))
	}

	cal, err := b.calendarRepo.GetByName(ctx, user.ID, calName)
	if err != nil {
		return nil, webdav.NewHTTPError(404, fmt.Errorf("calendar not found"))
	}

	// Encode iCalendar to bytes (this is what we persist)
	icsBytes, err := encodeICalendar(icalData)
	if err != nil {
		return nil, webdav.NewHTTPError(400, fmt.Errorf("failed to encode iCalendar: %w", err))
	}

	// Extract UID from iCalendar if not in path
	if uid == "" {
		uid = extractUIDFromICalendar(icalData)
	}
	if uid == "" {
		return nil, webdav.NewHTTPError(400, fmt.Errorf("no UID in iCalendar data"))
	}

	// Generate ETag from the ICS bytes we're persisting
	etag := domain.GenerateETag(icsBytes)

	// Extract metadata for SQL queries
	summary, dtStart, dtEnd, rrule, sequence := extractEventMetadata(icalData)

	// Check if event exists
	existing, err := b.eventRepo.GetByUID(ctx, cal.ID, uid)
	isNew := err == domain.ErrNotFound

	if isNew {
		// Creating new event
		// Check If-None-Match: * (client expects resource to not exist)
		if opts != nil && opts.IfNoneMatch.IsSet() && !opts.IfNoneMatch.IsWildcard() {
			return nil, webdav.NewHTTPError(412, fmt.Errorf("resource exists"))
		}

		event := &domain.Event{
			CalendarID:     cal.ID,
			UID:            uid,
			ICS:            string(icsBytes),
			Summary:        summary,
			StartTime:      dtStart,
			EndTime:        dtEnd,
			RecurrenceRule: rrule,
			ETag:           etag,
			Sequence:       sequence,
			Status:         "CONFIRMED",
		}

		// Begin transaction for atomic event create + sync token bump
		tx, err := b.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback() // no-op if committed

		eventRepoTx := b.eventRepo.WithTx(tx)
		calendarRepoTx := b.calendarRepo.WithTx(tx)

		if err := eventRepoTx.Create(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to create event: %w", err)
		}

		// Increment calendar sync token (in same transaction)
		if _, err := calendarRepoTx.IncrementSyncToken(ctx, cal.ID); err != nil {
			return nil, fmt.Errorf("failed to increment sync token: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		slog.Info("caldav.event.created", "username", user.Username, "calendar", calName, "uid", uid, "etag", etag)
		return b.domainEventToCalDAV(event, urlPath), nil
	}

	// Updating existing event
	// Save the old ETag for precondition checking
	oldETag := existing.ETag

	// Check If-Match header (ETag validation)
	if opts != nil && opts.IfMatch.IsSet() {
		if !opts.IfMatch.IsWildcard() {
			clientETag := string(opts.IfMatch)
			if clientETag != oldETag {
				slog.Debug("caldav.conflict", "expected_etag", clientETag, "actual_etag", oldETag, "status", 412)
				return nil, webdav.NewHTTPError(412, fmt.Errorf("ETag mismatch"))
			}
		}
	}

	// Update the event
	existing.ICS = string(icsBytes)
	existing.Summary = summary
	existing.StartTime = dtStart
	existing.EndTime = dtEnd
	existing.RecurrenceRule = rrule
	existing.ETag = etag
	existing.Sequence = sequence

	// Begin transaction for atomic event update + sync token bump
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	eventRepoTx := b.eventRepo.WithTx(tx)
	calendarRepoTx := b.calendarRepo.WithTx(tx)

	if err := eventRepoTx.Update(ctx, existing, oldETag); err != nil {
		if err == domain.ErrPreconditionFailed {
			slog.Debug("caldav.conflict", "expected_etag", oldETag, "actual_etag", "changed", "status", 412)
			return nil, webdav.NewHTTPError(412, fmt.Errorf("concurrent modification"))
		}
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Increment calendar sync token (in same transaction)
	if _, err := calendarRepoTx.IncrementSyncToken(ctx, cal.ID); err != nil {
		return nil, fmt.Errorf("failed to increment sync token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("caldav.event.updated", "username", user.Username, "calendar", calName, "uid", uid, "etag", etag)
	return b.domainEventToCalDAV(existing, urlPath), nil
}

// DeleteCalendarObject removes an event
// Event deletion and sync token bump are atomic (single transaction).
func (b *Backend) DeleteCalendarObject(ctx context.Context, urlPath string) error {
	user := getUserFromContext(ctx)
	if user == nil {
		return fmt.Errorf("no authenticated user")
	}

	calName, uid := extractCalendarAndUID(urlPath)
	cal, err := b.calendarRepo.GetByName(ctx, user.ID, calName)
	if err != nil {
		return webdav.NewHTTPError(404, fmt.Errorf("calendar not found"))
	}

	// Begin transaction for atomic event delete + sync token bump
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	eventRepoTx := b.eventRepo.WithTx(tx)
	calendarRepoTx := b.calendarRepo.WithTx(tx)

	if err := eventRepoTx.Delete(ctx, cal.ID, uid); err != nil {
		if err == domain.ErrNotFound {
			return webdav.NewHTTPError(404, fmt.Errorf("event not found"))
		}
		return fmt.Errorf("failed to delete event: %w", err)
	}

	// Increment calendar sync token (in same transaction)
	if _, err := calendarRepoTx.IncrementSyncToken(ctx, cal.ID); err != nil {
		return fmt.Errorf("failed to increment sync token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("caldav.event.deleted", "username", user.Username, "calendar", calName, "uid", uid)
	return nil
}

// Helper functions

func (b *Backend) domainCalendarToCalDAV(cal *domain.Calendar, username string) caldav.Calendar {
	return caldav.Calendar{
		Path:                  fmt.Sprintf("/dav/calendars/%s/%s/", username, cal.Name),
		Name:                  cal.DisplayName,
		Description:           cal.Description,
		SupportedComponentSet: []string{"VEVENT"},
	}
}

func (b *Backend) domainEventToCalDAV(event *domain.Event, urlPath string) *caldav.CalendarObject {
	// Parse ICS string back to ical.Calendar
	icalCal, err := parseICalendar(event.ICS)
	if err != nil {
		slog.Error("failed to parse stored ICS", "error", err, "uid", event.UID)
		// Return empty calendar on error to avoid crashes
		icalCal = ical.NewCalendar()
	}

	return &caldav.CalendarObject{
		Path:    urlPath,
		ModTime: event.UpdatedAt,
		ETag:    event.ETag,
		Data:    icalCal,
	}
}

// Context key for user
type contextKey string

const userContextKey contextKey = "user"

func getUserFromContext(ctx context.Context) *domain.User {
	if user, ok := ctx.Value(userContextKey).(*domain.User); ok {
		return user
	}
	return nil
}

// SetUserInContext adds user to context (used by auth middleware)
func SetUserInContext(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// Path parsing helpers
// Note: go-webdav's caldav.Handler strips the Prefix ("/dav") before calling backend methods,
// so we receive paths like "/calendars/{user}/{cal}/" not "/dav/calendars/{user}/{cal}/".
// These helpers handle both forms for robustness.

func extractCalendarName(urlPath string) string {
	// Handles both:
	//   /dav/calendars/{username}/{calendar-name}/ -> calendar-name
	//   /calendars/{username}/{calendar-name}/     -> calendar-name (prefix stripped)
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")

	// Find the "calendars" segment and extract calendar name relative to it
	for i, part := range parts {
		if part == "calendars" && i+2 < len(parts) {
			return parts[i+2] // calendar name is 2 positions after "calendars"
		}
	}
	return ""
}

func extractCalendarAndUID(urlPath string) (calName, uid string) {
	// Handles both:
	//   /dav/calendars/{username}/{calendar}/{uid}.ics -> calendar, uid
	//   /calendars/{username}/{calendar}/{uid}.ics     -> calendar, uid (prefix stripped)
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")

	// Find the "calendars" segment and extract relative to it
	for i, part := range parts {
		if part == "calendars" && i+2 < len(parts) {
			calName = parts[i+2] // calendar name
			if i+3 < len(parts) {
				uid = strings.TrimSuffix(parts[i+3], ".ics")
			}
			return
		}
	}
	return
}

func ensureTrailingSlash(s string) string {
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}
	return s
}

// ICS helper functions using go-ical library

// encodeICalendar encodes an ical.Calendar to bytes
func encodeICalendar(cal *ical.Calendar) ([]byte, error) {
	var buf bytes.Buffer
	encoder := ical.NewEncoder(&buf)
	if err := encoder.Encode(cal); err != nil {
		return nil, fmt.Errorf("failed to encode ical: %w", err)
	}
	return buf.Bytes(), nil
}

// parseICalendar parses an iCalendar string into an ical.Calendar
func parseICalendar(icsData string) (*ical.Calendar, error) {
	decoder := ical.NewDecoder(strings.NewReader(icsData))
	cal, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode ical: %w", err)
	}
	return cal, nil
}

// extractUIDFromICalendar extracts the UID from the first VEVENT component
func extractUIDFromICalendar(cal *ical.Calendar) string {
	for _, comp := range cal.Children {
		if comp.Name == "VEVENT" {
			if uidProp := comp.Props.Get("UID"); uidProp != nil {
				return uidProp.Value
			}
		}
	}
	return ""
}

// extractEventMetadata extracts metadata from VEVENT for SQL storage
func extractEventMetadata(cal *ical.Calendar) (summary string, dtStart, dtEnd time.Time, rrule string, sequence int) {
	for _, comp := range cal.Children {
		if comp.Name == "VEVENT" {
			// Extract SUMMARY
			if prop := comp.Props.Get("SUMMARY"); prop != nil {
				summary = prop.Value
			}

			// Extract DTSTART
			if prop := comp.Props.Get("DTSTART"); prop != nil {
				dtStart, _ = parseICalTime(prop)
			}

			// Extract DTEND or calculate from DURATION
			if prop := comp.Props.Get("DTEND"); prop != nil {
				dtEnd, _ = parseICalTime(prop)
			} else if prop := comp.Props.Get("DURATION"); prop != nil {
				// Parse duration and add to dtStart
				duration, err := parseICalDuration(prop.Value)
				if err == nil {
					dtEnd = dtStart.Add(duration)
				}
			}

			// Extract RRULE
			if prop := comp.Props.Get("RRULE"); prop != nil {
				rrule = prop.Value
			}

			// Extract SEQUENCE
			if prop := comp.Props.Get("SEQUENCE"); prop != nil {
				seq, err := strconv.Atoi(prop.Value)
				if err == nil {
					sequence = seq
				}
			}

			// Only process the first VEVENT
			break
		}
	}
	return
}

// parseICalTime parses an iCalendar DATE-TIME or DATE property
func parseICalTime(prop *ical.Prop) (time.Time, error) {
	value := prop.Value

	// Check for TZID parameter
	tzid := prop.Params.Get("TZID")

	// Try different formats
	formats := []string{
		"20060102T150405Z",     // UTC format
		"20060102T150405",      // Local/floating format
		"20060102",             // DATE format
		"2006-01-02T15:04:05Z", // ISO 8601 UTC
		"2006-01-02T15:04:05",  // ISO 8601 local
		"2006-01-02",           // ISO 8601 date
	}

	var t time.Time
	var err error

	for _, format := range formats {
		t, err = time.Parse(format, value)
		if err == nil {
			break
		}
	}

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time %q: %w", value, err)
	}

	// If TZID is specified, try to load the timezone
	if tzid != "" {
		loc, err := time.LoadLocation(tzid)
		if err == nil {
			// Re-interpret the time in the specified timezone
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
		}
	}

	return t, nil
}

// parseICalDuration parses an iCalendar DURATION value (e.g., "PT1H30M")
func parseICalDuration(value string) (time.Duration, error) {
	// Basic DURATION parser - supports PT[nH][nM][nS] format
	// This is a simplified implementation; full RFC 5545 support would be more complex

	if !strings.HasPrefix(value, "P") {
		return 0, fmt.Errorf("invalid duration format: %s", value)
	}

	value = strings.TrimPrefix(value, "P")
	isTime := strings.HasPrefix(value, "T")
	if isTime {
		value = strings.TrimPrefix(value, "T")
	}

	var duration time.Duration

	// Parse hours
	if idx := strings.Index(value, "H"); idx != -1 {
		hours, err := strconv.Atoi(value[:idx])
		if err != nil {
			return 0, fmt.Errorf("invalid hours in duration: %w", err)
		}
		duration += time.Duration(hours) * time.Hour
		value = value[idx+1:]
	}

	// Parse minutes
	if idx := strings.Index(value, "M"); idx != -1 {
		minutes, err := strconv.Atoi(value[:idx])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes in duration: %w", err)
		}
		duration += time.Duration(minutes) * time.Minute
		value = value[idx+1:]
	}

	// Parse seconds
	if idx := strings.Index(value, "S"); idx != -1 {
		seconds, err := strconv.Atoi(value[:idx])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds in duration: %w", err)
		}
		duration += time.Duration(seconds) * time.Second
	}

	return duration, nil
}
