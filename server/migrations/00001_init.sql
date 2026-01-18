-- +goose Up
-- Initial schema for Calendar-app
-- Single-user MVP; designed for future Postgres migration

-- Users table (single user for MVP, but structured for expansion)
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Calendars table (CalDAV calendar collections)
CREATE TABLE IF NOT EXISTS calendars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    color TEXT,
    description TEXT,
    sync_token TEXT,  -- CalDAV sync-token for incremental sync
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Enforce uniqueness: one calendar per (user_id, name) combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendars_user_id_name ON calendars(user_id, name);

-- Events table (iCalendar VEVENT components)
-- Hybrid storage: full ICS for CalDAV fidelity + extracted metadata for queries
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    calendar_id INTEGER NOT NULL,
    uid TEXT NOT NULL,  -- iCalendar UID (globally unique)
    ics TEXT NOT NULL,  -- Full VEVENT component (stored as-is for CalDAV roundtrip)
    -- Extracted metadata for efficient SQL queries:
    summary TEXT,  -- Event title
    description TEXT,
    location TEXT,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    all_day BOOLEAN DEFAULT 0,
    recurrence_rule TEXT,  -- RRULE for recurring events
    etag TEXT NOT NULL,  -- SHA-256(ics) for conflict detection
    sequence INTEGER DEFAULT 0,  -- iCalendar SEQUENCE
    status TEXT DEFAULT 'CONFIRMED',  -- TENTATIVE, CONFIRMED, CANCELLED
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (calendar_id) REFERENCES calendars(id) ON DELETE CASCADE,
    UNIQUE(calendar_id, uid)
);

CREATE INDEX idx_events_calendar_id ON events(calendar_id);
CREATE INDEX idx_events_uid ON events(uid);
CREATE INDEX idx_events_start_time ON events(start_time);
CREATE INDEX idx_events_end_time ON events(end_time);

-- Tasks table (Todoist sync data + VTODO components)
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    todoist_id TEXT UNIQUE,  -- Todoist task ID (nullable for local tasks)
    content TEXT NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 1,
    due_date DATETIME,
    completed BOOLEAN DEFAULT 0,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_todoist_id ON tasks(todoist_id);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);

-- Preferences table (user configuration for AI planning)
CREATE TABLE IF NOT EXISTS preferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value TEXT,  -- JSON-encoded value
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, key)
);

CREATE INDEX idx_preferences_user_id ON preferences(user_id);

-- Audit log table (plan apply history for rollback)
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    action TEXT NOT NULL,  -- 'plan_apply', 'event_create', etc.
    entity_type TEXT,  -- 'event', 'task', etc.
    entity_id TEXT,
    changes TEXT,  -- JSON-encoded before/after state
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- +goose Down
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS preferences;
DROP TABLE IF EXISTS tasks;
DROP INDEX IF EXISTS idx_events_end_time;
DROP INDEX IF EXISTS idx_events_start_time;
DROP INDEX IF EXISTS idx_events_uid;
DROP INDEX IF EXISTS idx_events_calendar_id;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS calendars;
DROP TABLE IF EXISTS users;
