package caldav

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/airplne/calendar-app/server/internal/domain"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Username string
	Password string
}

// LoadAuthConfig loads auth configuration from environment
func LoadAuthConfig() AuthConfig {
	username := os.Getenv("CALENDARAPP_USER")
	password := os.Getenv("CALENDARAPP_PASS")

	if username == "" {
		username = "testuser"
		slog.Warn("Using default username - set CALENDARAPP_USER in production")
	}
	if password == "" {
		password = "testpass"
		slog.Warn("Using default password - set CALENDARAPP_PASS in production")
	}

	return AuthConfig{
		Username: username,
		Password: password,
	}
}

// BasicAuthMiddleware creates HTTP Basic auth middleware
func BasicAuthMiddleware(config AuthConfig, userRepo domain.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()

			if !ok || username != config.Username || password != config.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="CalDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get user from database
			user, err := userRepo.GetByUsername(r.Context(), username)
			if err != nil {
				slog.Error("Failed to get user for auth", "username", username, "error", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user to context
			ctx := SetUserInContext(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Note: SetUserInContext, GetUserFromContext, and context key helpers
// are defined in backend.go to avoid circular dependencies
