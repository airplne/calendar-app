# PRP: CalDAV Trust, Sync Health, and Interop Gate

**Type:** Implementation PRP / Product Requirements Prompt  
**Target Team:** Backend + Web + QA/Interop  
**Status:** Proposed  
**Priority:** MVP foundation

---

## 1. Context

Calendar-app's product wedge is a trustworthy self-hosted CalDAV source of truth before AI planning complexity. The app should prove multi-client calendar safety before asking users to connect Todoist or LLM providers.

This PRP strengthens the current CalDAV foundation with sync observability, privacy-safe debug exports, manual interop evidence, and an onboarding gate that requires green CalDAV sync before Todoist/LLM setup.

Current repo assumptions:

- Backend: Go, Chi, emersion/go-webdav, SQLite, goose, slog.
- Frontend: React, Vite, Tailwind, TanStack Query/Zustand.
- CalDAV mount: `/dav`.
- Raw ICS is canonical for CalDAV roundtrip fidelity.
- ETags are derived from stored ICS bytes.
- Corrupt stored ICS must fail visibly, not silently return empty data.

---

## 2. Problem statement

Users cannot trust AI planning until they trust that Calendar-app safely syncs their calendar across real CalDAV clients. Self-hosters also need a way to diagnose sync failures without leaking private calendar content into logs or debug bundles.

Calendar-app needs an explicit "CalDAV works before AI" gate: users should prove create/edit/delete from Apple Calendar, Fantastical, DAVx5, or another CalDAV client before Todoist or LLM setup becomes the next step.

---

## 3. Goals

- Prove CalDAV interoperability with at least two primary clients before first AI setup.
- Add a Sync Health dashboard that exposes recent CalDAV operations without raw event content.
- Add a redacted debug bundle export for self-hosted support.
- Document manual interop results in `docs/testing/caldav-interop-results.md`.
- Add onboarding that guides users to green sync before Todoist/LLM setup.
- Strengthen data-safety checks around roundtrip fidelity, stale ETags, duplicate UIDs, and corrupt ICS.
- Make sync failures diagnosable by client fingerprint, operation type, status, and redacted error.

---

## 4. Non-goals

- No real LLM work.
- No Todoist setup before green CalDAV sync.
- No full CalDAV scheduling/iTIP.
- No full RFC 6578 `sync-collection` unless a concrete client interop blocker is discovered.
- No hosted multi-tenant admin console.
- No raw ICS/event descriptions in default logs, dashboard rows, or debug bundles.
- No broad observability platform dependency.

---

## 5. User stories

1. As a self-hosting user, I can see the correct CalDAV URL and app-specific password instructions, connect Apple Calendar or DAVx5, and run a validation event to prove sync works.
2. As a beta user, I can read a repo doc showing which CalDAV clients pass create/edit/delete testing and known quirks.
3. As an operator, I can open Sync Health and see recent PROPFIND/REPORT/GET/PUT/DELETE operations with user-agent fingerprints, status codes, ETag outcomes, and durations.
4. As a privacy-conscious user, I can export a debug bundle that excludes raw ICS, event descriptions, attendee names, and task details by default.
5. As a user, I get a visible error when stored ICS is corrupt; the server never returns a false successful empty calendar.
6. As a user, I can see that a stale ETag write was rejected with HTTP 412 and which client caused it.

---

## 6. Functional requirements

### FR1 - CalDAV interop evidence

Create or update `docs/testing/caldav-interop-results.md` with:

- Test date.
- Calendar-app commit SHA/version.
- Server OS/runtime.
- Client name/version.
- Client platform.
- Account discovery status.
- Create event status.
- Read/list status.
- Edit event status.
- Delete event status.
- ETag conflict behavior.
- Duplicate UID behavior.
- Basic recurrence behavior.
- Timezone event behavior.
- Known quirks.
- Screenshots/log snippets if useful, redacted.

Required clients:

- Apple Calendar.
- Fantastical.
- DAVx5.
- Thunderbird if easy; mark as `not tested` or `deferred` if not available.

Pass threshold:

- At least 2 of Apple Calendar, Fantastical, and DAVx5 must pass manual CRUD.

### FR2 - Sync operation logging

Add structured metadata logging for CalDAV requests.

Capture these fields:

- `operation_id`.
- `timestamp`.
- `method`: `PROPFIND`, `REPORT`, `GET`, `PUT`, `DELETE`, `PROPPATCH`.
- `route_template` or redacted path shape.
- `status_code`.
- `duration_ms`.
- `user_id`.
- `calendar_id`, if known.
- `event_uid_hash`, if known.
- `client_user_agent`.
- `client_fingerprint`.
- `etag_present`.
- `etag_match_result`: `matched`, `stale`, `missing`, `not_applicable`.
- `duplicate_uid_attempt`.
- `parse_failure`.
- `corrupt_ics_incident`.
- `error_code`.
- `redacted_error_message`.
- `request_size_bytes`.
- `response_size_bytes`, optional.

Redaction rules:

- Do not log raw ICS.
- Do not log event descriptions.
- Do not log attendee names/emails.
- Do not log full path segments that may contain user-supplied event names.
- Hash UID and path identifiers where needed.

### FR3 - Sync Health dashboard

Add a web UI page, likely `/sync-health`, showing:

- Overall status: Healthy / Warning / Critical / Unknown.
- Last successful CalDAV operation.
- Last failed CalDAV operation.
- Last successful sync validation event.
- Last failed sync validation event.
- Client fingerprints/user agents seen in last 7 days.
- Recent 50 CalDAV operations.
- ETag conflict count, last 24h and all-time.
- Duplicate UID attempts, last 24h and all-time.
- Parse failures, last 24h and all-time.
- Corrupt ICS incidents, last 24h and all-time.
- Median and p95 operation duration.
- Links to export debug bundle, interop results, and client setup instructions.

Recent operation table columns:

- Time.
- Client.
- Method.
- Resource type: calendar/event/principal/unknown.
- Status.
- ETag result.
- Duration.
- Redacted error.

### FR4 - Sync Health API

Add REST endpoints:

```http
GET  /api/v1/sync-health
GET  /api/v1/sync-health/operations?limit=50
GET  /api/v1/sync-health/clients
POST /api/v1/sync-health/validation-event/start
POST /api/v1/sync-health/validation-event/verify
GET  /api/v1/debug-bundle
```

Example `GET /api/v1/sync-health` response:

```json
{
  "status": "healthy",
  "last_success_at": "2026-04-25T14:05:02Z",
  "last_failure_at": null,
  "operation_counts_24h": {
    "total": 128,
    "success": 126,
    "failure": 2,
    "etag_conflicts": 1,
    "duplicate_uid_attempts": 0,
    "parse_failures": 0,
    "corrupt_ics_incidents": 0
  },
  "latency_ms": {
    "median": 22,
    "p95": 180
  },
  "clients": [
    {
      "fingerprint": "apple-calendar-macos-redacted",
      "display_name": "Apple Calendar",
      "last_seen_at": "2026-04-25T14:00:00Z",
      "operation_count_24h": 44
    }
  ]
}
```

### FR5 - Debug bundle export

Add a debug bundle export as `.zip` or `.tar.gz`.

Default contents:

- `manifest.json` with Calendar-app version/commit, runtime summary, config summary with secrets redacted, data directory presence, database migration version.
- `sync_health_summary.json`.
- `recent_caldav_operations.json`.
- `client_fingerprints.json`.
- `error_traces.log`.
- `schema_summary.txt`.
- `interop_checklist_template.md`.

Default exclusions:

- Raw ICS.
- Event descriptions.
- Attendees.
- Task content.
- LLM API keys.
- Todoist tokens.
- Passwords.
- Authorization headers.
- Session cookies.

Optional explicit toggle:

```http
GET /api/v1/debug-bundle?include_raw_event_data=true
```

Rules:

- UI must display a warning before raw event data inclusion.
- Raw data export must require explicit user action.
- Include manifest flag: `"raw_event_data_included": true`.

### FR6 - Onboarding gate

Add onboarding sequence:

1. Show server status.
2. Show CalDAV URL: `https://<host>/dav/`, principal URL if needed, and calendar home URL pattern if needed.
3. Generate or display app-specific CalDAV password.
4. Show client setup recipes for Apple Calendar, Fantastical, DAVx5, and optionally Thunderbird.
5. Start validation event flow:
   - User creates event from external client.
   - User edits event title/time from external client.
   - User deletes event from external client.
6. Calendar-app detects each step by operation log and repository state.
7. Mark `Green sync` only when create/edit/delete pass.
8. Unlock Todoist setup.
9. Unlock LLM setup.

Validation event:

- Suggested event title: `Calendar-app Sync Test`.
- Use generated UID prefix or marker to detect safely.
- Delete the test event after validation, or require external client delete step.

Failure states must include explanation, last observed operation, suggested next step, debug bundle link, and client-specific guidance.

### FR7 - Data safety checks

Automated and runtime behavior must guarantee:

- PUT -> GET roundtrip preserves event data.
- ETag conflict handling returns HTTP 412 on stale writes.
- Duplicate UID attempts do not create duplicate events.
- Corrupt stored ICS returns visible 500/diagnostic error, not empty success.
- Parse failures increment metrics and appear in Sync Health.
- PUT/DELETE mutate event and calendar sync token atomically.
- UID uniqueness remains enforced at DB level.
- Any parse or storage failure is logged with redacted metadata.

---

## 7. Non-functional requirements

- Privacy: no raw event content in logs or default debug bundles.
- Reliability: sync health instrumentation must not break CalDAV request handling.
- Performance: logging overhead should add less than 10ms p95 for typical CalDAV operations.
- Durability: operation logs should survive restarts with bounded retention.
- Retention: default keep recent operation metadata for 7 days or last 1,000 operations, whichever is smaller.
- Accessibility: Sync Health and onboarding UI must meet WCAG 2.1 AA.
- Local-first: diagnostics are generated locally by the self-hosted instance.
- Graceful degradation: if metrics persistence fails, CalDAV operations continue and log a redacted warning.

---

## 8. Technical approach

### Backend

Add an instrumentation layer around the CalDAV handler/middleware:

- Generate `operation_id` at request start.
- Capture method, path shape, user agent, start time, response status, duration.
- Extract safe authenticated user/calendar/event identifiers where possible.
- Capture ETag/precondition outcomes from backend errors where possible.
- Persist operation metadata to SQLite.

Add explicit domain errors:

- `ErrDuplicateUID`.
- `ErrCorruptICS`.
- `ErrParseFailed`.
- `ErrStaleETag`.
- `ErrInteropValidationFailed`.

Suggested service interfaces:

```go
type SyncHealthService interface {
    RecordOperation(ctx context.Context, op CalDAVOperation) error
    Summary(ctx context.Context) (*SyncHealthSummary, error)
    RecentOperations(ctx context.Context, limit int) ([]CalDAVOperation, error)
    Clients(ctx context.Context) ([]ClientFingerprint, error)
    StartValidationEvent(ctx context.Context, userID int64) (*ValidationSession, error)
    VerifyValidationEvent(ctx context.Context, sessionID string) (*ValidationResult, error)
}

type DebugBundleService interface {
    Build(ctx context.Context, opts DebugBundleOptions) (*DebugBundle, error)
}
```

### Frontend

Add routes/pages:

- `/onboarding/caldav`.
- `/sync-health`.
- `/settings/debug`.

Use TanStack Query for:

- Sync health summary.
- Recent operations.
- Debug bundle download.
- Validation session state.

---

## 9. Files likely to change

Backend:

```text
server/internal/caldav/handler.go
server/internal/caldav/backend.go
server/internal/caldav/auth.go
server/internal/caldav/proppatch.go
server/internal/caldav/sync.go
server/internal/caldav/observability.go
server/internal/caldav/client_fingerprint.go
server/internal/api/sync_health.go
server/internal/api/debug_bundle.go
server/internal/data/sqlite_caldav_operation_repo.go
server/internal/data/sqlite_validation_repo.go
server/internal/domain/sync_health.go
server/internal/domain/errors.go
server/internal/services/sync_health.go
server/internal/services/debug_bundle.go
server/migrations/00002_sync_health.sql
server/cmd/calendarapp/main.go
```

Frontend:

```text
web/src/pages/OnboardingCalDAVPage.tsx
web/src/pages/SyncHealthPage.tsx
web/src/pages/DebugBundlePage.tsx
web/src/components/sync/SyncStatusBadge.tsx
web/src/components/sync/OperationTable.tsx
web/src/components/sync/ClientList.tsx
web/src/components/onboarding/CalDAVSetupRecipe.tsx
web/src/lib/api.ts
web/src/lib/types.ts
web/src/App.tsx
```

Docs/tests:

```text
docs/testing/caldav-interop-results.md
docs/setup/caldav-clients.md
server/internal/caldav/*_test.go
server/internal/api/sync_health_test.go
server/internal/services/debug_bundle_test.go
```

---

## 10. API changes

```http
GET  /api/v1/sync-health
GET  /api/v1/sync-health/operations?limit=50
GET  /api/v1/sync-health/clients
POST /api/v1/sync-health/validation-event/start
POST /api/v1/sync-health/validation-event/verify
GET  /api/v1/debug-bundle
GET  /api/v1/debug-bundle?include_raw_event_data=true
```

---

## 11. Database/schema changes

Create migration `server/migrations/00002_sync_health.sql`.

```sql
CREATE TABLE caldav_operations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  operation_id TEXT NOT NULL UNIQUE,
  user_id INTEGER,
  calendar_id INTEGER,
  event_uid_hash TEXT,
  method TEXT NOT NULL,
  route_shape TEXT NOT NULL,
  status_code INTEGER NOT NULL,
  duration_ms INTEGER NOT NULL,
  client_user_agent TEXT,
  client_fingerprint TEXT,
  etag_present BOOLEAN DEFAULT 0,
  etag_match_result TEXT,
  duplicate_uid_attempt BOOLEAN DEFAULT 0,
  parse_failure BOOLEAN DEFAULT 0,
  corrupt_ics_incident BOOLEAN DEFAULT 0,
  error_code TEXT,
  redacted_error_message TEXT,
  request_size_bytes INTEGER,
  response_size_bytes INTEGER,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
  FOREIGN KEY (calendar_id) REFERENCES calendars(id) ON DELETE SET NULL
);

CREATE INDEX idx_caldav_operations_created_at ON caldav_operations(created_at);
CREATE INDEX idx_caldav_operations_user_id ON caldav_operations(user_id);
CREATE INDEX idx_caldav_operations_client_fingerprint ON caldav_operations(client_fingerprint);
CREATE INDEX idx_caldav_operations_status_code ON caldav_operations(status_code);

CREATE TABLE sync_validation_sessions (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL,
  expected_uid_prefix TEXT NOT NULL,
  status TEXT NOT NULL,
  create_observed_at DATETIME,
  edit_observed_at DATETIME,
  delete_observed_at DATETIME,
  failure_code TEXT,
  failure_message TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

---

## 12. UX changes

### Onboarding checklist

```text
CalDAV setup

[ ] Server is running
[ ] App-specific password created
[ ] Connect one calendar client
[ ] Create validation event
[ ] Edit validation event
[ ] Delete validation event

Todoist and LLM setup unlock after Green Sync.
```

### Sync Health summary

```text
Sync: Healthy
Last successful sync: 2 minutes ago
Clients seen: Apple Calendar, DAVx5
ETag conflicts today: 1
Parse failures today: 0
Corrupt ICS incidents: 0
p95 operation duration: 180ms
```

---

## 13. Test plan

Automated tests:

- `TestCalDAVOperationLoggedOnPUT`.
- `TestCalDAVOperationLoggedOnGET`.
- `TestCalDAVOperationLogsNoRawICS`.
- `TestSyncHealthSummaryCountsETagConflicts`.
- `TestDebugBundleExcludesRawICSByDefault`.
- `TestDebugBundleExcludesEventDescriptionsByDefault`.
- `TestDebugBundleIncludesRawDataOnlyWithExplicitOption`.
- `TestValidationEventCreateEditDeleteFlow`.
- `TestStaleETagReturns412AndIncrementsConflictCount`.
- `TestDuplicateUIDRejectedAndLogged`.
- `TestCorruptStoredICSReturns500AndLogged`.
- `TestPUTGETRoundtripPreservesICS`.

Manual tests:

- Apple Calendar create/edit/delete.
- Fantastical create/edit/delete.
- DAVx5 create/edit/delete.
- Thunderbird optional create/edit/delete.
- Simultaneous edit from two clients.
- Debug bundle privacy review.

---

## 14. Manual validation steps

1. Run `cd server && go test ./...`.
2. Start the server.
3. Open onboarding.
4. Generate app-specific password.
5. Add Apple Calendar account using `/dav/`.
6. Create `Calendar-app Sync Test`.
7. Confirm UI detects create.
8. Edit test event.
9. Confirm UI detects edit.
10. Delete test event.
11. Confirm UI shows green sync.
12. Open Sync Health.
13. Confirm recent operations show methods/status/duration/client without raw content.
14. Export default debug bundle.
15. Inspect bundle for raw ICS/event descriptions; none should exist.
16. Repeat with DAVx5 or Fantastical.
17. Update `docs/testing/caldav-interop-results.md`.

---

## 15. Acceptance criteria

- `go test ./...` passes.
- PUT -> GET roundtrip preserves event data.
- Stale ETag writes return HTTP 412.
- At least 2 of Apple Calendar, Fantastical, and DAVx5 pass manual CRUD testing.
- Sync Health page shows latest client operations without leaking raw event content.
- Debug bundle exports redacted diagnostics.
- Debug bundle raw event data requires explicit user opt-in.
- Onboarding can verify create/edit/delete from at least one external CalDAV client.
- Todoist/LLM setup is visually and functionally gated behind green sync.
- `docs/testing/caldav-interop-results.md` exists and is updated.
- Corrupt stored ICS fails visibly and increments corrupt incident count.
- Duplicate UID attempts are rejected and visible in Sync Health.

---

## 16. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| Logs leak private data | Critical | Redaction tests, bundle inspection tests, no raw ICS by default |
| Instrumentation breaks CalDAV | High | Middleware-only logging, fail-open if logging persistence fails |
| Client-specific quirks consume scope | Medium | Document quirks; require 2 of 3 primary clients, Thunderbird optional |
| Validation event false negatives | Medium | Use operation log + repository state; provide manual retry |
| Debug bundle grows large | Low | Retention limits, compressed bundle, metadata-only default |
| Sync Health creates anxiety | Low | Explain statuses and next actions clearly |

---

## 17. Open questions

- Should app-specific CalDAV passwords be implemented in this PRP or can dev env Basic Auth remain for the first version?
- What retention default is best: 7 days, last 1,000 operations, or both?
- Should Sync Health be visible before login for local single-user setups, or authenticated only?
- Should client fingerprinting normalize known clients into friendly names?
- Should validation event run against the default calendar only?

---

## 18. Suggested GitHub issue breakdown

1. `feat(server): add CalDAV operation metadata logging`
2. `feat(server): add sync health repository and summary service`
3. `feat(api): expose sync health endpoints`
4. `feat(server): add redacted debug bundle export`
5. `feat(web): add Sync Health dashboard`
6. `feat(web): add CalDAV onboarding gate`
7. `test(caldav): add validation-event flow tests`
8. `test(caldav): add stale ETag, duplicate UID, corrupt ICS diagnostics tests`
9. `docs(testing): add CalDAV interop results matrix`
10. `docs(setup): add Apple Calendar, Fantastical, DAVx5 setup recipes`
