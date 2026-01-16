package caldav

import (
	"fmt"
	"net/http"
)

// NewHandler returns the CalDAV/WebDAV HTTP handler mounted at /dav.
//
// Bootstrap note: this is intentionally a stub. Replace this with a real CalDAV
// handler backed by emersion/go-webdav (and its CalDAV support) once the data
// layer and authentication are in place.
func NewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintln(w, "CalDAV endpoint not implemented yet. TODO: integrate emersion/go-webdav.")
	})
}

