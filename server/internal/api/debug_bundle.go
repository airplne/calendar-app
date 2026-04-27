package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/airplne/calendar-app/server/internal/services"
)

type DebugBundleHandler struct {
	service *services.DebugBundleService
}

func NewDebugBundleHandler(service *services.DebugBundleService) *DebugBundleHandler {
	return &DebugBundleHandler{service: service}
}

func (h *DebugBundleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	bundle, err := h.service.Build(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "debug_bundle_unavailable", "Debug bundle is unavailable.")
		return
	}

	actor := bundle.GeneratedBy
	if actor == "" {
		actor = "authenticated-user"
	}
	slog.Info("debug_bundle.exported", "timestamp", time.Now().UTC(), "actor", actor, "redaction_mode", bundle.Redaction.Mode)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="calendar-app-debug-bundle.json"`)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(bundle)
}
