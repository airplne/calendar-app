# internal/api

REST API handlers for Calendar-app-specific features.

## Responsibilities

- AI planning endpoints (`/api/v1/plan/*`)
- Todoist sync endpoints (`/api/v1/todoist/*`)
- User preferences (`/api/v1/preferences`)
- Audit log (`/api/v1/audit/*`)
- Integration API (`/api/v1/me/focus-status`)

## Key Files (to be created)

- `router.go` - Chi router setup for REST endpoints
- `plan.go` - AI planning request handlers
- `todoist.go` - Todoist sync trigger handlers
- `preferences.go` - User preference CRUD
- `middleware.go` - Auth, logging, CORS middleware
