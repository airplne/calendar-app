-- +goose Up
-- Redacted CalDAV operation metadata for Sync Health diagnostics.
-- Stores request metadata only; never stores raw ICS or event descriptions.

CREATE TABLE IF NOT EXISTS caldav_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation_id TEXT NOT NULL UNIQUE,
    occurred_at DATETIME NOT NULL,
    method TEXT NOT NULL,
    path_pattern TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    client_user_agent TEXT,
    client_fingerprint TEXT NOT NULL,
    etag_outcome TEXT NOT NULL,
    operation_kind TEXT NOT NULL,
    outcome TEXT NOT NULL,
    error_code TEXT,
    redacted_error TEXT,
    request_size_bytes INTEGER DEFAULT 0,
    response_size_bytes INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_caldav_operations_occurred_at ON caldav_operations(occurred_at);
CREATE INDEX IF NOT EXISTS idx_caldav_operations_method ON caldav_operations(method);
CREATE INDEX IF NOT EXISTS idx_caldav_operations_client_fingerprint ON caldav_operations(client_fingerprint);
CREATE INDEX IF NOT EXISTS idx_caldav_operations_status_code ON caldav_operations(status_code);
CREATE INDEX IF NOT EXISTS idx_caldav_operations_outcome ON caldav_operations(outcome);
CREATE INDEX IF NOT EXISTS idx_caldav_operations_error_code ON caldav_operations(error_code);

-- +goose Down
DROP INDEX IF EXISTS idx_caldav_operations_error_code;
DROP INDEX IF EXISTS idx_caldav_operations_outcome;
DROP INDEX IF EXISTS idx_caldav_operations_status_code;
DROP INDEX IF EXISTS idx_caldav_operations_client_fingerprint;
DROP INDEX IF EXISTS idx_caldav_operations_method;
DROP INDEX IF EXISTS idx_caldav_operations_occurred_at;
DROP TABLE IF EXISTS caldav_operations;
