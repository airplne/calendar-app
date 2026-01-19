package caldav

import (
	"encoding/xml"
	"log/slog"
	"net/http"
)

// DAV XML namespace
const davNS = "DAV:"

// PropPatchHandler handles PROPPATCH requests for Apple Calendar compatibility
// Returns HTTP 207 Multi-Status with all properties marked as successfully set
// (This is a no-op - we don't actually modify properties, but Apple Calendar requires this response)
type PropPatchHandler struct{}

func NewPropPatchHandler() *PropPatchHandler {
	return &PropPatchHandler{}
}

// multistatus is the root element of a 207 response
type multistatus struct {
	XMLName   xml.Name   `xml:"DAV: multistatus"`
	Responses []response `xml:"response"`
}

type response struct {
	Href     string   `xml:"href"`
	PropStat propstat `xml:"propstat"`
}

type propstat struct {
	Prop   prop   `xml:"prop"`
	Status string `xml:"status"`
}

type prop struct {
	// We echo back whatever properties the client sent
	// For simplicity, just mark them all as successful
}

func (h *PropPatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PROPPATCH" {
		// Not a PROPPATCH request, pass through
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Debug("PROPPATCH request (no-op)", "path", r.URL.Path)

	// Build 207 Multi-Status response
	// We mark all property updates as successful even though we don't actually store them
	resp := multistatus{
		Responses: []response{
			{
				Href: r.URL.Path,
				PropStat: propstat{
					Prop:   prop{},
					Status: "HTTP/1.1 200 OK",
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus) // 207

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(resp); err != nil {
		slog.Error("Failed to encode PROPPATCH response", "error", err)
	}
}
