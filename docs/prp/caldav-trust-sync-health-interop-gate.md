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
- MVP CalDAV auth uses the existing HTTP Basic Auth / environment credential path.

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
- Keep MVP auth simple by reusing the existing Basic Auth/env credential path.

---

## 4. Non-goals

- No real LLM work.
- No Todoist setup before green CalDAV sync.
- No full CalDAV scheduling/iTIP.
- No full RFC 6578 `sync-collection` unless a concrete client interop blocker is discovered.
- No hosted multi-tenant admin console.
- No raw ICS/event descriptions in default logs, dashboard rows, or debug bundles.
- No broad observability platform dependency.
- No new credential model in this PRP.
- No app-specific password generation, password rotation, or per-client credentials in MVP.

---

## 5. MVP Auth Decision

For MVP, Calendar-app will reuse the existing HTTP Basic Auth / environment credential path, currently configured through `CALENDARAPP_USER` and `CALENDARAPP_PASS`.

The onboarding UI should display:

- configured CalDAV username instructions,
- configured password instructions,
- server URL,
- principal/calendar URL guidance where needed,
- client setup recipes for supported CalDAV clients.

The onboarding UI should not create, rotate, or manage per-client app-specific passwords in this PRP.

App-specific CalDAV passwords are a post-MVP enhancement and should be tracked separately. A future credential PRP should define schema, generation endpoint, rotation/revocation UX, per-client labels, and tests.

---

## 6. User stories

1. As a self-hosting user, I can see the correct CalDAV URL and Basic Auth credential instructions, connect Apple Calendar or DAVx5, and run a validation event to prove sync works.
2. As a beta user, I can read a repo doc showing which CalDAV clients pass create/edit/delete testing and known quirks.
3. As an operator, I can open Sync Health and see recent PROPFIND/REPORT/GET/PUT/DELETE operations with client fingerprints, status codes, ETag outcomes, and durations.
4. As a privacy-conscious user, I can export a debug bundle that excludes raw ICS, event descriptions, attendee names, and task details by default.
5. As a user, I get a visible error when stored ICS is corrupt; the server never returns a false successful empty calendar.
6. As a user, I can see that a stale ETag write was rejected with HTTP 412 and which client caused it.

---

## 7. Functional requirements

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

- Apple Calendar, Fantastical, and DAVx5 must all be represented in `docs/testing/caldav-interop-results.md`.
- At least 2 of Apple Calendar, Fantastical, and DAVx5 must pass manual CRUD for MVP.
- Non-passing or not-yet-tested clients must be documented with status, reason, and follow-up issue.

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
- Raw `client_user_agent` may be stored locally for self-hosted diagnostics, but exported debug bundles should normalize or redact unusually identifying user-agent values by default.

Debug bundles may include normalized client fingerprints such as:

- `apple-calendar`.
- `fantastical`.
- `davx5`.
- `thunderbird`.
- `unknown-caldav-client`.

Raw user-agent export requires explicit user opt-in.

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

### FR4 - Sync Health status rules

Status computation must be deterministic. Exact thresholds should be implementation-configurable, but MVP defaults must be documented and tested.

`Healthy`

- Green-sync validation completed successfully.
- Last validation event create/edit/delete passed.
- No CalDAV write failures in the recent operation window.
- No corrupt ICS incidents in the recent operation window.
- No unresolved duplicate UID incidents.
- ETag conflicts, if present, are expected HTTP 412 conflicts and below warning threshold.

`Warning`

- Green-sync validation completed previously but recent operations show recoverable failures.
- Recent ETag conflicts exceed threshold.
- A supported client has recent sync failures but data integrity is not known to be compromised.
- Validation is stale and should be rerun.

`Critical`

- Corrupt stored ICS detected.
- Duplicate UID incident unresolved.
- PUT/GET roundtrip validation failed.
- Green-sync validation failed.
- Calendar write path is failing.
- Data integrity may be at risk.

`Unknown`

- Green-sync validation has not been completed.
- No recent client operation data exists.
- Server cannot determine sync health.

MVP default thresholds:

- Recent operation window: latest 50 operations or last 24 hours, whichever has data.
- Warning threshold for expected ETag conflicts: more than 5 stale-write conflicts in the recent operation window.
- Validation stale threshold: green-sync validation older than 14 days.
- Any corrupt ICS incident in the recent operation window is `Critical` until resolved or explicitly acknowledged by future tooling.
- Any unresolved duplicate UID incident is `Critical` until resolved.

### FR5 - Sync Health API

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
  "green_sync_completed": true,
  "green_sync_completed_at": "2026-04-25T13:50:00Z",
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
      "fingerprint": "apple-calendar",
      "display_name": "Apple Calendar",
      "last_seen_at": "2026-04-25T14:00:00Z",
      "operation_count_24h": 44
    }
  ]
}
```

### FR6 - Debug bundle export

Add a debug bundle export as `.zip` or `.tar.gz`.

`/api/v1/debug-bundle` must require authenticated local admin/session access. It must never be publicly accessible.

Debug bundle export should be auditable in local logs with timestamp and authenticated user/session identifier, but without raw event content.

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
- Raw user-agent strings when they look unusually identifying.

Optional explicit toggle:

```http
GET /api/v1/debug-bundle?include_raw_event_data=true
```

Rules:

- UI must display a warning before raw event data inclusion.
- Raw data export must require explicit user action.
- Include manifest flag: `"raw_event_data_included": true`.
- Raw user-agent export requires explicit user opt-in, separate from normalized fingerprint export.

### FR7 - Onboarding gate

Add onboarding sequence:

1. Show server status.
2. Show CalDAV URL: `https://<host>/dav/`, principal URL if needed, and calendar home URL pattern if needed.
3. Display configured CalDAV username/password instructions from the existing Basic Auth/env credential path.
4. Show client setup recipes for Apple Calendar, Fantastical, DAVx5, and optionally Thunderbird.
5. Start validation event flow:
   - User creates event from external client.
   - User edits event title/time from external client.
   - User deletes event from external client.
6. Calendar-app detects each step by operation log and repository state.
7. Mark `Green sync` only when create/edit/delete pass.
8. Unlock Todoist setup.
9. Unlock LLM setup.

MVP credential scope:

- Do not create a new credential model.
- Do not add password-generation endpoints.
- Do not add password-rotation UI.
- Do not add per-client credentials.
- App-specific CalDAV passwords are post-MVP.

Validation event:

- Suggested event title: `Calendar-app Sync Test`.
- Use generated UID prefix or marker to detect safely.
- Delete the test event after validation, or require external client delete step.

Failure states must include explanation, last observed operation, suggested next step, debug bundle link, and client-specific guidance.

### FR8 - Data safety checks

Automated and runtime behavior must guarantee:

- PUT -> GET roundtrip preserves event data.
- ETag conflict handling returns HTTP 412 on stale writes.
- Duplicate UID attempts do not create duplicate events.
- Corrupt stored ICS returns visible 500/diagnostic error, not empty success.
- Parse failures increment metrics and appear in Sync Health.
- PUT/DELETE mutate event and calendar sync token atomically.
- UID uniqueness remains enforced at DB level.
- Any parse or storage failure is logged with redacted metadata.

### FR9 - Retention cleanup

Sync operation logs must have a bounded retention policy.

MVP default:

- retain the latest 1,000 operations, or
- retain 14 days of operations,
- whichever limit is reached first.

A cleanup job or write-time pruning must enforce this retention.

---

## 8. Non-functional requirements

- Privacy: no raw event content in logs or default debug bundles.
- Reliability: sync health instrumentation must not break CalDAV request handling.
- Performance: logging overhead should add less than 10ms p95 for typical CalDAV operations.
- Durability: operation logs should survive restarts with bounded retention.
- Retention: latest 1,000 operations or 14 days, whichever limit is reached first.
- Accessibility: Sync Health and onboarding UI must meet WCAG 2.1 AA.
- Local-first: diagnostics are generated locally by the self-hosted instance.
- Auth: debug bundle access requires authenticated local admin/session access.
- Graceful degradation: if metrics persistence fails, CalDAV operations continue and log a redacted warning.

---

## 9. Technical approach

### Backend

Add an instrumentation layer around the CalDAV handler/middleware:

- Generate `operation_id` at request start.
- Capture method, path shape, normalized client fingerprint, start time, response status, duration.
- Store raw user agent locally only when useful for self-hosted diagnostics.
- Extract safe authenticated user/calendar/event identifiers where possible.
- Capture ETag/precondition outcomes from backend errors where possible.
- Persist operation metadata to SQLite.
- Enforce retention by cleanup job or write-time pruning.

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
    PruneOperations(ctx context.Context, policy RetentionPolicy) error
}

type DebugBundleService interface {
    Build(ctx context.Context, userID int64, opts DebugBundleOptions) (*DebugBundle, error)
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

## 10. Files likely to change

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

## 11. API changes

```http
GET  /api/v1/sync-health
GET  /api/v1/sync-health/operations?limit=50
GET  /api/v1/sync-health/clients
POST /api/v1/sync-health/validation-event/start
POST /api/v1/sync-health/validation-event/verify
GET  /api/v1/debug-bundle
GET  /api/v1/debug-bundle?include_raw_event_data=true
```

No credential-generation API is included in MVP.

---

## 12. Database/schema changes

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

No credential schema changes are required in this PRP.

---

## 13. UX changes

### Onboarding checklist

```text
CalDAV setup

[ ] Server is running
[ ] CalDAV URL displayed
[ ] Basic Auth username/password instructions displayed
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

## 14. Test plan

Automated tests:

- `TestCalDAVOperationLoggedOnPUT`.
- `TestCalDAVOperationLoggedOnGET`.
- `TestCalDAVOperationLogsNoRawICS`.
- `TestSyncHealthSummaryCountsETagConflicts`.
- `TestSyncHealthStatusHealthy`.
- `TestSyncHealthStatusWarning`.
- `TestSyncHealthStatusCritical`.
- `TestSyncHealthStatusUnknown`.
- `TestDebugBundleRequiresAuthenticatedAccess`.
- `TestDebugBundleExcludesRawICSByDefault`.
- `TestDebugBundleExcludesEventDescriptionsByDefault`.
- `TestDebugBundleNormalizesClientUserAgentsByDefault`.
- `TestDebugBundleIncludesRawDataOnlyWithExplicitOption`.
- `TestValidationEventCreateEditDeleteFlow`.
- `TestStaleETagReturns412AndIncrementsConflictCount`.
- `TestDuplicateUIDRejectedAndLogged`.
- `TestCorruptStoredICSReturns500AndLogged`.
- `TestPUTGETRoundtripPreservesICS`.
- `TestOperationRetentionPrunesOldRows`.

Manual tests:

- Apple Calendar create/edit/delete.
- Fantastical create/edit/delete.
- DAVx5 create/edit/delete.
- Thunderbird optional create/edit/delete.
- Simultaneous edit from two clients.
- Debug bundle privacy review.

---

## 15. Manual validation steps

1. Run `cd server && go test ./...`.
2. Start the server with `CALENDARAPP_USER` and `CALENDARAPP_PASS` configured.
3. Open onboarding.
4. Confirm the CalDAV URL and Basic Auth credential instructions are displayed.
5. Add Apple Calendar account using `/dav/` and the configured Basic Auth credentials.
6. Create `Calendar-app Sync Test`.
7. Confirm UI detects create.
8. Edit test event.
9. Confirm UI detects edit.
10. Delete test event.
11. Confirm UI shows green sync.
12. Open Sync Health.
13. Confirm recent operations show methods/status/duration/client without raw content.
14. Export default debug bundle while authenticated.
15. Inspect bundle for raw ICS/event descriptions/raw identifying UA; none should exist by default.
16. Repeat with DAVx5 or Fantastical.
17. Update `docs/testing/caldav-interop-results.md`.

---

## 16. Acceptance criteria

- `go test ./...` passes.
- PUT -> GET roundtrip preserves event data.
- Stale ETag writes return HTTP 412.
- Apple Calendar, Fantastical, and DAVx5 are all represented in `docs/testing/caldav-interop-results.md`.
- At least 2 of Apple Calendar, Fantastical, and DAVx5 pass manual CRUD testing.
- Sync Health page shows latest client operations without leaking raw event content.
- Sync Health statuses use deterministic Healthy / Warning / Critical / Unknown rules.
- Debug bundle endpoint requires authenticated local admin/session access.
- Debug bundle exports redacted diagnostics by default.
- Debug bundle raw event data requires explicit user opt-in.
- Raw user-agent export requires explicit user opt-in.
- Onboarding can verify create/edit/delete from at least one external CalDAV client.
- Onboarding uses existing Basic Auth/env credentials and does not generate app-specific passwords.
- Todoist/LLM setup is visually and functionally gated behind green sync.
- `docs/testing/caldav-interop-results.md` exists and is updated.
- Corrupt stored ICS fails visibly and increments corrupt incident count.
- Duplicate UID attempts are rejected and visible in Sync Health.
- Operation log retention is enforced by cleanup job or write-time pruning.

---

## 17. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| Logs leak private data | Critical | Redaction tests, bundle inspection tests, no raw ICS by default |
| Debug bundle is exposed without auth | Critical | Require authenticated local admin/session access and audit bundle export |
| Instrumentation breaks CalDAV | High | Middleware-only logging, fail-open if logging persistence fails |
| Client-specific quirks consume scope | Medium | Document quirks; require 2 of 3 primary clients, Thunderbird optional |
| Validation event false negatives | Medium | Use operation log + repository state; provide manual retry |
| Operation logs grow without bound | Medium | Retention cleanup: latest 1,000 operations or 14 days |
| Debug bundle grows large | Low | Retention limits, compressed bundle, metadata-only default |
| Sync Health creates anxiety | Low | Explain statuses and next actions clearly |

---

## 18. Open questions

- Should Sync Health be visible before login for local single-user setups, or authenticated only?
- Should client fingerprinting normalize additional clients beyond Apple Calendar, Fantastical, DAVx5, and Thunderbird?
- Should validation event run against the default calendar only?
- Should post-MVP app-specific CalDAV passwords be per-device, per-client, or both?

---

## 19. Suggested GitHub issue breakdown

1. `feat(server): add CalDAV operation metadata logging`
2. `feat(server): add sync health repository and summary service`
3. `feat(api): expose sync health endpoints`
4. `feat(server): add redacted debug bundle export`
5. `feat(web): add Sync Health dashboard`
6. `feat(web): add CalDAV onboarding gate using Basic Auth/env credentials`
7. `test(caldav): add validation-event flow tests`
8. `test(caldav): add stale ETag, duplicate UID, corrupt ICS diagnostics tests`
9. `test(sync): add health status and retention tests`
10. `docs(testing): add CalDAV interop results matrix`
11. `docs(setup): add Apple Calendar, Fantastical, DAVx5 setup recipes`
12. `post-mvp: design app-specific CalDAV password model`
