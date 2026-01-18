package caldav

import (
	"net/http"
)

// WellKnownHandler handles /.well-known/caldav requests
// Redirects to the CalDAV principal URL for auto-discovery
type WellKnownHandler struct {
	// PrincipalPath is the path to redirect to
	// For MVP single-user: /dav/principals/testuser/
	PrincipalPath string
}

func NewWellKnownHandler(principalPath string) *WellKnownHandler {
	return &WellKnownHandler{
		PrincipalPath: principalPath,
	}
}

func (h *WellKnownHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Return 301 redirect to principal URL
	// This helps CalDAV clients discover the correct URL
	http.Redirect(w, r, h.PrincipalPath, http.StatusMovedPermanently)
}
