package caldav

import (
	"net/http"

	"github.com/emersion/go-webdav/caldav"
	"github.com/go-chi/chi/v5"

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
	calendarRepo domain.CalendarRepo
	eventRepo    domain.EventRepo
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

// NewHandlerWithRepos creates the real CalDAV handler with repository access
func NewHandlerWithRepos(userRepo domain.UserRepo, calendarRepo domain.CalendarRepo, eventRepo domain.EventRepo) http.Handler {
	authConfig := LoadAuthConfig()

	backend := NewBackend(userRepo, calendarRepo, eventRepo)

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

	// Mount go-webdav handler for all CalDAV methods
	r.Handle("/*", caldavHandler)

	return r
}

// NewWellKnownRoutes returns a handler for /.well-known/caldav
// The username parameter determines the principal redirect target
func NewWellKnownRoutes(username string) http.Handler {
	return NewWellKnownHandler("/dav/principals/" + username + "/")
}
