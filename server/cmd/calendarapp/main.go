package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/airplne/calendar-app/server/internal/caldav"
	"github.com/airplne/calendar-app/server/internal/data"
	"github.com/airplne/calendar-app/server/internal/domain"
	"github.com/airplne/calendar-app/server/internal/webui"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var migrateOnly = flag.Bool("migrate-only", false, "Run database migrations and exit")

const (
	defaultPort          = "8080"
	defaultDataDir       = "./data"
	defaultMigrationsDir = "./migrations"
)

func main() {
	flag.Parse()

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration from environment
	port := getEnv("CALENDARAPP_PORT", defaultPort)
	dataDir := getEnv("CALENDARAPP_DATA_DIR", defaultDataDir)
	migrationsDir := getEnv("CALENDARAPP_MIGRATIONS_DIR", defaultMigrationsDir)

	slog.Info("Starting Calendar-app server",
		"port", port,
		"data_dir", dataDir,
		"migrations_dir", migrationsDir,
	)

	// Open database
	db, err := data.OpenDB(dataDir)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer data.CloseDB(db)

	// Run migrations
	if err := data.RunMigrations(db, migrationsDir); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Handle --migrate-only flag
	if *migrateOnly {
		slog.Info("Migrations complete, exiting")
		return
	}

	// Initialize repositories
	userRepo := data.NewSQLiteUserRepo(db)
	calendarRepo := data.NewSQLiteCalendarRepo(db)
	eventRepo := data.NewSQLiteEventRepo(db)

	// Load auth config to get configured username
	authConfig := caldav.LoadAuthConfig()

	// Ensure configured user exists (MVP single-user mode)
	ctx := context.Background()
	user, err := userRepo.GetByUsername(ctx, authConfig.Username)
	if errors.Is(err, domain.ErrNotFound) {
		user, err = userRepo.Create(ctx, authConfig.Username)
		if err != nil {
			slog.Error("Failed to create default user", "error", err)
			os.Exit(1)
		}
		slog.Info("Created default user", "username", authConfig.Username)
	}

	// Ensure default calendar exists for user
	_, err = calendarRepo.GetByName(ctx, user.ID, "default")
	if errors.Is(err, domain.ErrNotFound) {
		defaultCal := &domain.Calendar{
			UserID:      user.ID,
			Name:        "default",
			DisplayName: "Calendar",
		}
		if err := calendarRepo.Create(ctx, defaultCal); err != nil {
			slog.Error("Failed to create default calendar", "error", err)
			os.Exit(1)
		}
		slog.Info("Created default calendar for user", "username", authConfig.Username)
	}

	// Initialize router (Chi per locked MVP decisions)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Health endpoint
	r.Get("/health", handleHealth)

	// SSE endpoint stub
	r.Get("/events", handleSSE)

	// Well-known CalDAV auto-discovery endpoint
	r.Get("/.well-known/caldav", caldav.NewWellKnownRoutes(authConfig.Username).ServeHTTP)

	// CalDAV mount point with repository access
	r.Mount("/dav", caldav.NewHandlerWithRepos(userRepo, calendarRepo, eventRepo))

	// Web UI (embedded in production; placeholder when dist not built)
	r.Mount("/", webui.Handler())

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	// SSE endpoint stub - will implement real-time push updates
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	// Keep connection alive (stub - will add real event broadcasting)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
