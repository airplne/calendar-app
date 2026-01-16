# internal/caldav

CalDAV/WebDAV protocol implementation.

## Responsibilities

- Wrap `emersion/go-webdav` library
- Implement backend interfaces for calendar/event storage
- Handle PROPPATCH no-op for Apple Calendar compatibility
- Manage CalDAV sync tokens and ETags

## Key Files (to be created)

- `handler.go` - CalDAV HTTP handler setup
- `backend.go` - Bridge between go-webdav and our data layer
- `auth.go` - CalDAV-specific authentication (HTTP Basic/Digest)
