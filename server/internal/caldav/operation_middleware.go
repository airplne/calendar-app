package caldav

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

// OperationMetadataMiddleware records redacted CalDAV request metadata. It must
// not inspect or persist request/response bodies.
func OperationMetadataMiddleware(recorder domain.CalDAVOperationRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if recorder == nil {
				next.ServeHTTP(w, r)
				return
			}

			started := time.Now()
			wrapped := &operationResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			operation := BuildCalDAVOperation(
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				time.Since(started),
				r.Header,
				wrapped.Header(),
				r.UserAgent(),
				r.ContentLength,
				wrapped.bytesWritten,
			)
			if err := recorder.Record(&operation); err != nil {
				// Recording must never break CalDAV request handling.
				slog.Warn("failed to record CalDAV operation metadata", "error", err)
			}
		})
	}
}

type operationResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *operationResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *operationResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)
	return n, err
}
