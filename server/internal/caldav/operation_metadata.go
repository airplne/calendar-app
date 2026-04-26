package caldav

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

func NormalizeClientFingerprint(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "fantastical"):
		return domain.CalDAVClientFantastical
	case strings.Contains(ua, "davx5") || strings.Contains(ua, "davx⁵"):
		return domain.CalDAVClientDAVx5
	case strings.Contains(ua, "thunderbird"):
		return domain.CalDAVClientThunderbird
	case strings.Contains(ua, "dataaccess") || strings.Contains(ua, "calendarstore") || strings.Contains(ua, "apple"):
		return domain.CalDAVClientAppleCalendar
	default:
		return domain.CalDAVClientUnknown
	}
}

func RedactCalDAVPath(rawPath string) string {
	if rawPath == "" {
		return "/"
	}
	clean := path.Clean(rawPath)
	if clean == "/." {
		clean = "/"
	}
	if clean == "/.well-known/caldav" {
		return "/.well-known/caldav"
	}
	if clean == "/dav" || clean == "/dav/" {
		return "/dav/"
	}

	parts := strings.Split(strings.Trim(clean, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "/"
	}

	for i, part := range parts {
		if part == "calendars" {
			prefix := ""
			if i > 0 && parts[0] == "dav" {
				prefix = "/dav"
			}
			if len(parts) == i+1 {
				return prefix + "/calendars/"
			}
			if len(parts) == i+2 {
				return prefix + "/calendars/{principal}/"
			}
			if len(parts) == i+3 {
				return prefix + "/calendars/{principal}/{calendar}/"
			}
			return prefix + "/calendars/{principal}/{calendar}/{object}.ics"
		}
		if part == "principals" {
			prefix := ""
			if i > 0 && parts[0] == "dav" {
				prefix = "/dav"
			}
			return prefix + "/principals/{principal}/"
		}
	}

	if parts[0] == "dav" {
		return "/dav/{resource}"
	}
	return "/{resource}"
}

func ClassifyOperationKind(method string) domain.CalDAVOperationKind {
	switch strings.ToUpper(method) {
	case http.MethodPut, http.MethodDelete, "PROPPATCH", "MKCOL", "MOVE", "COPY", http.MethodPost, http.MethodPatch:
		return domain.CalDAVOperationWrite
	default:
		return domain.CalDAVOperationRead
	}
}

func ClassifyETagOutcome(method string, statusCode int, requestHeader http.Header, responseHeader http.Header) domain.CalDAVETagOutcome {
	if statusCode == http.StatusPreconditionFailed {
		return domain.CalDAVETagMismatched
	}
	if requestHeader.Get("If-Match") != "" {
		return domain.CalDAVETagMatched
	}
	if requestHeader.Get("If-None-Match") != "" {
		return domain.CalDAVETagMissing
	}
	if responseHeader.Get("ETag") != "" {
		return domain.CalDAVETagGenerated
	}
	return domain.CalDAVETagNotApplicable
}

func ClassifyOperationOutcome(kind domain.CalDAVOperationKind, statusCode int, errCode domain.CalDAVErrorCode) domain.CalDAVOperationOutcome {
	if statusCode >= 200 && statusCode < 400 && errCode == domain.CalDAVErrorNone {
		return domain.CalDAVOperationSuccess
	}
	if errCode == domain.CalDAVErrorCorruptICS || errCode == domain.CalDAVErrorDuplicateUID || errCode == domain.CalDAVErrorParse {
		return domain.CalDAVOperationIntegrityFailure
	}
	if statusCode >= 500 {
		return domain.CalDAVOperationIntegrityFailure
	}
	return domain.CalDAVOperationRecoverableFailure
}

func RedactedCalDAVError(statusCode int, method string) (domain.CalDAVErrorCode, string) {
	if statusCode >= 200 && statusCode < 400 {
		return domain.CalDAVErrorNone, ""
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return domain.CalDAVErrorAuthFailed, "Authentication failed."
	}
	if statusCode == http.StatusPreconditionFailed {
		return domain.CalDAVErrorETagConflict, "ETag precondition failed."
	}
	if statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusNotImplemented {
		return domain.CalDAVErrorUnsupportedMethod, "Unsupported CalDAV method."
	}
	if statusCode >= 500 {
		return domain.CalDAVErrorWriteFailed, "CalDAV operation failed."
	}
	if strings.EqualFold(method, http.MethodPut) || strings.EqualFold(method, "PROPPATCH") {
		return domain.CalDAVErrorWriteFailed, "CalDAV write failed."
	}
	return domain.CalDAVErrorUnknown, "CalDAV operation failed."
}

func BuildCalDAVOperation(method string, rawPath string, statusCode int, duration time.Duration, requestHeader http.Header, responseHeader http.Header, userAgent string, requestSize int64, responseSize int64) domain.CalDAVOperation {
	kind := ClassifyOperationKind(method)
	errorCode, redactedError := RedactedCalDAVError(statusCode, method)
	etagOutcome := ClassifyETagOutcome(method, statusCode, requestHeader, responseHeader)
	if errorCode == domain.CalDAVErrorETagConflict {
		etagOutcome = domain.CalDAVETagMismatched
	}

	return domain.CalDAVOperation{
		ID:                fmt.Sprintf("caldav-%d", time.Now().UnixNano()),
		OccurredAt:        time.Now().UTC(),
		Method:            strings.ToUpper(method),
		PathPattern:       RedactCalDAVPath(rawPath),
		StatusCode:        statusCode,
		DurationMillis:    duration.Milliseconds(),
		ClientUserAgent:   userAgent,
		ClientFingerprint: NormalizeClientFingerprint(userAgent),
		ETagOutcome:       etagOutcome,
		OperationKind:     kind,
		Outcome:           ClassifyOperationOutcome(kind, statusCode, errorCode),
		ErrorCode:         errorCode,
		RedactedError:     redactedError,
		RequestSizeBytes:  requestSize,
		ResponseSizeBytes: responseSize,
	}
}
