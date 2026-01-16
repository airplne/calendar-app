# internal/data

Data access layer for SQLite database.

## Responsibilities

- Database connection management (SQLite with WAL mode)
- Event/calendar/task models
- Repository interfaces for data operations
- Migration runner (goose integration)
- Schema designed for future Postgres migration

## Key Files (to be created)

- `db.go` - Database connection setup
- `models.go` - Data model definitions
- `events.go` - Event repository
- `calendars.go` - Calendar repository
- `tasks.go` - Task repository (Todoist sync data)
- `migrations.go` - Migration runner logic
