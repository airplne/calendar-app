package caldav

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/emersion/go-webdav/caldav"
	"github.com/go-chi/chi/v5"

	"github.com/airplne/calendar-app/server/internal/data"
	"github.com/airplne/calendar-app/server/internal/domain"
)

func init() {
	// Register WebDAV/CalDAV methods not in Chi's standard set
	chi.RegisterMethod("PROPFIND")
	chi.RegisterMethod("PROPPATCH")
	chi.RegisterMethod("REPORT")
	chi.RegisterMethod("MKCOL")
	chi.RegisterMethod("MOVE")
	chi.RegisterMethod("COPY")
}

// Handler creates the CalDAV HTTP handler with all routes
type Handler struct {
	backend      *Backend
	authConfig   AuthConfig
	userRepo     domain.UserRepo
	calendarRepo *data.SQLiteCalendarRepo
	eventRepo    *data.SQLiteEventRepo
}

// NewHandler creates a new CalDAV handler
// For backward compatibility during migration, accepts no args and returns stub
// Use NewHandlerWithRepos for the real implementation
func NewHandler() http.Handler {
	// Stub for backward compatibility - will be replaced
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("CalDAV endpoint not implemented yet. Use NewHandlerWithRepos."))
	})
}

// NewHandlerWithRepos creates the real CalDAV handler with repository access.
// The db parameter is required for transaction support (atomic event write + sync token bump).
// The calendarRepo and eventRepo must be concrete SQLite repos to support WithTx.
func NewHandlerWithRepos(db *sql.DB, userRepo domain.UserRepo, calendarRepo *data.SQLiteCalendarRepo, eventRepo *data.SQLiteEventRepo) http.Handler {
	authConfig := LoadAuthConfig()

	backend := NewBackend(db, userRepo, calendarRepo, eventRepo)

	// Create go-webdav CalDAV handler
	caldavHandler := &caldav.Handler{
		Backend: backend,
		Prefix:  "/dav",
	}

	// Create Chi router for CalDAV routes
	r := chi.NewRouter()

	// Apply Basic Auth middleware
	r.Use(BasicAuthMiddleware(authConfig, userRepo))

	// PROPPATCH interception middleware (Apple Calendar compatibility)
	// Must be before caldavHandler since r.Handle("/*") would catch all methods
	proppatchHandler := NewPropPatchHandler()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == "PROPPATCH" {
				proppatchHandler.ServeHTTP(w, req)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// Content-Type hardening middleware for .ics GET responses
	// Ensures proper Content-Type for CalDAV clients sensitive to header values
	r.Use(ICSContentTypeMiddleware)

	// Mount go-webdav handler for all CalDAV methods
	r.Handle("/*", caldavHandler)

	return r
}

// NewWellKnownRoutes returns a handler for /.well-known/caldav
// The username parameter determines the principal redirect target
func NewWellKnownRoutes(username string) http.Handler {
	return NewWellKnownHandler("/dav/principals/" + username + "/")
}

// ICSContentTypeMiddleware ensures GET requests for .ics files return proper Content-Type.
// Some CalDAV clients are sensitive to missing/incorrect Content-Type headers.
func ICSContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only intercept GET requests for .ics files
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, ".ics") {
			// Wrap ResponseWriter to set Content-Type before first write
			wrapped := &icsResponseWriter{ResponseWriter: w, headerSet: false}
			next.ServeHTTP(wrapped, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// icsResponseWriter wraps http.ResponseWriter to ensure Content-Type is set for .ics responses
type icsResponseWriter struct {
	http.ResponseWriter
	headerSet bool
}

func (w *icsResponseWriter) WriteHeader(code int) {
	w.ensureContentType()
	w.ResponseWriter.WriteHeader(code)
}

func (w *icsResponseWriter) Write(data []byte) (int, error) {
	w.ensureContentType()
	return w.ResponseWriter.Write(data)
}

func (w *icsResponseWriter) ensureContentType() {
	if w.headerSet {
		return
	}
	w.headerSet = true
	// Ensure proper Content-Type with charset for .ics files
	// Some CalDAV clients are sensitive to missing charset
	ct := w.ResponseWriter.Header().Get("Content-Type")
	if ct == "" || ct == "text/calendar" {
		w.ResponseWriter.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	}
}
