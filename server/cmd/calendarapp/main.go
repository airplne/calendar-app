package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed ../../web/dist
var webDist embed.FS

const (
	defaultPort    = "8080"
	defaultDataDir = "./data"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration from environment
	port := getEnv("CALENDARAPP_PORT", defaultPort)
	dataDir := getEnv("CALENDARAPP_DATA_DIR", defaultDataDir)

	slog.Info("Starting Calendar-app server",
		"port", port,
		"data_dir", dataDir,
	)

	// Initialize router
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", handleHealth)

	// SSE endpoint stub
	mux.HandleFunc("/events", handleSSE)

	// Serve embedded web UI (in production) or proxy to Vite (in dev)
	// For now, serve a simple message until web/dist is built
	mux.HandleFunc("/", handleRoot)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
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

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Try to serve from embedded dist
	distFS, err := fs.Sub(webDist, "../../web/dist")
	if err != nil {
		// Fallback: serve a simple message
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Calendar-app</title></head>
<body>
  <h1>Calendar-app Server</h1>
  <p>Backend is running. Build the frontend with <code>cd web && npm run build</code></p>
  <p>Health check: <a href="/health">/health</a></p>
</body>
</html>`)
		return
	}

	// Serve from embedded filesystem
	fileServer := http.FileServer(http.FS(distFS))
	fileServer.ServeHTTP(w, r)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
