# PRP: MVP Implementation Sequence — CalDAV Repository + Planning Skeleton

**Type:** Implementation Prompt
**Target Team:** Claude Dev Team
**Date:** 2026-01-16
**Author:** Daniel

---

## 1. Context / Goal

**Calendar-app** is an AI planning co-pilot built on a self-hosted CalDAV server. The MVP validates two hypotheses:
1. **Problem-solving:** Does ambient AI protection reduce burnout for meeting-heavy knowledge workers?
2. **Experience:** Is self-hosted ownership + data sovereignty compelling vs. cloud alternatives?

### Current State

**Completed:**
- ✅ PRD with 65 FRs across 12 capability areas (`_bmad-output/planning-artifacts/prd.md`)
- ✅ PRD Validation Report (PASS) (`_bmad-output/planning-artifacts/prd-validation-report.md`)
- ✅ Monorepo skeleton on `main`: `server/` (Go) + `web/` (React)

**Stack Decisions (Locked):**
- **Backend:** Go 1.22+, Chi router, emersion/go-webdav, SQLite (glebarez/CGO-free), goose migrations, slog logging
- **Frontend:** React 18 + Vite (SPA), HashRouter, Tailwind CSS, headless UI primitives (copy-paste), TanStack Query + Zustand
- **Real-time:** Server-Sent Events (SSE) at `/events`
- **Deployment:** Single binary (embed.FS) + Docker (distroless)

### Implementation Goal

Execute a **phased MVP sequence** prioritizing CalDAV multi-client sync correctness before building the AI planning layer:

**Phase A:** Domain model + repository layer with proper abstractions
**Phase B:** CalDAV CRUD + ETag conflict handling; validate with 3 clients
**Phase C:** Constraint validation engine + planning API stub

This approach ensures the calendar foundation is solid (zero data loss, reliable sync) before adding AI features. If AI quality is poor, the CalDAV server still provides value.

### References

| Document | Path | Purpose |
|----------|------|---------|
| PRD | `_bmad-output/planning-artifacts/prd.md` | FRs/NFRs, user journeys |
| PRD Validation | `_bmad-output/planning-artifacts/prd-validation-report.md` | PASS validation report |
| Repo README | `README.md` | Current stack + local dev workflow |
| GPT-5 Research | `docs/prp/gpt-5-pro-research-starters-and-monorepo.md` | Stack research input |

---

## 2. Out of Scope (for this PRP)

The following are **explicitly excluded** from MVP implementation:

| Feature | RFC/Standard | Why Deferred |
|---------|--------------|--------------|
| **CalDAV Scheduling** | RFC 6638 (iTIP invites) | "Defend/reschedule" = draft messages only; calendar changes reflect after organizer updates invite + CalDAV sync |
| **Full AI implementation** | N/A | Phase C creates stub only; real LLM integration comes after CalDAV validation |
| **Todoist integration** | N/A | Comes after CalDAV sync proven reliable |
| **Advanced recurrence** | RFC 5545 (complex RRULE) | MVP supports basic daily/weekly; defer nth-weekday, exceptions |
| **Multi-user/multi-tenant** | N/A | Single-user MVP; architecture shouldn't block, but don't implement |
| **WebDAV sync-collection (REPORT)** | RFC 6578 | Defer to post-MVP; keep `calendars.sync_token` as an internal revision token (don’t rely on client incremental sync) |
| **Mobile clients** | N/A | Test with DAVx⁵, but no mobile app development |
| **Web UI polish** | N/A | Placeholder pages sufficient; focus on API correctness |

**Critical Scope Boundaries:**

- **CalDAV operations:** Implement minimal subset for CRUD + basic sync only
- **Constraint validation:** Deterministic checker for hard constraints only (time conflicts, overlaps); soft constraints (energy patterns, preferences) come later
- **Planning API:** Stub returning mock proposal data; no real LLM calls yet
- **Auth:** Simple token-based or HTTP Basic; no OAuth, no user management UI

---

## 3. Phased Work Plan

### Phase A: Domain Model + Repository Layer

**Goal:** Establish clean abstractions for event/calendar data with testability and future Postgres migration in mind.

**Deliverables:**

| Component | File Path | Responsibilities |
|-----------|-----------|------------------|
| Domain models | `server/internal/domain/event.go` | Go structs for Event, Calendar, Task with validation methods |
| Repository interfaces | `server/internal/domain/repository.go` | Interfaces: `EventRepo`, `CalendarRepo`, `TaskRepo` (data access contracts) |
| SQLite implementation | `server/internal/data/sqlite_event_repo.go` | Concrete SQLite implementation of repository interfaces |
| DB connection | `server/internal/data/db.go` | SQLite connection with WAL mode, migration runner |
| Migration runner | `server/internal/data/migrations.go` | Goose integration; auto-run on startup or via CLI flag |

**Key Design Decisions:**

1. **Event Storage Format:**
   ```go
   type Event struct {
       ID           int64
       CalendarID   int64
       UID          string  // iCalendar UID (globally unique)
       ICS          string  // Full iCalendar VEVENT component (stored as-is)
       // Extracted metadata for querying
       Summary      string
       StartTime    time.Time
       EndTime      time.Time
       AllDay       bool
       RecurrenceRule string  // RRULE string if recurring
       ETag         string  // SHA-256 hash of ICS for conflict detection
       Sequence     int     // iCalendar SEQUENCE for versioning
       LastModified time.Time
   }
   ```

2. **Repository Pattern:**
   ```go
   type EventRepo interface {
       Create(ctx context.Context, event *Event) error
       GetByUID(ctx context.Context, calendarID int64, uid string) (*Event, error)
       List(ctx context.Context, calendarID int64, start, end time.Time) ([]*Event, error)
       Update(ctx context.Context, event *Event, expectedETag string) error  // Returns ErrPreconditionFailed if ETag mismatch
       Delete(ctx context.Context, calendarID int64, uid string) error
   }
   ```

3. **ETag Generation:**
   - `ETag = SHA256(ICS)` - deterministic hash of the full iCalendar component
   - Stored in `events.etag` column
   - Used for HTTP If-Match/If-None-Match precondition checks

4. **Transaction Handling:**
   - All mutations wrapped in SQLite transactions
   - Use `database/sql.Tx` for atomicity
   - Rollback on constraint violations or errors

**Acceptance Criteria:**

```bash
# Must pass:
cd server && go test ./internal/domain/...
cd server && go test ./internal/data/...

# Migration runs without error:
cd server && go run ./cmd/calendarapp --migrate-only

# Database initialized with correct schema:
sqlite3 data/calendar.db ".schema events"  # Should match 00001_init.sql
```

**Definition of Done:**
- [ ] Domain models defined with validation methods
- [ ] Repository interfaces documented with Go interfaces
- [ ] SQLite implementation passes unit tests (create, read, update with ETag check, delete)
- [ ] Migration runner auto-runs on startup (creates DB if missing)
- [ ] ETag generation tested (same ICS = same ETag)
- [ ] Concurrent write test passes (simulated conflict returns error)

---

### Phase B: Minimal CalDAV CRUD + Multi-Client Sync

**Goal:** Implement CalDAV protocol handlers using `emersion/go-webdav`, validate multi-client sync correctness with Fantastical, Apple Calendar, and DAVx⁵.

**Deliverables:**

| Component | File Path | Responsibilities |
|-----------|-----------|------------------|
| CalDAV backend adapter | `server/internal/caldav/backend.go` | Implements `caldav.Backend` interface from go-webdav; bridges to our EventRepo |
| CalDAV handler | `server/internal/caldav/handler.go` | Mounts go-webdav handler at `/dav`; configures auth |
| PROPPATCH handler | `server/internal/caldav/proppatch.go` | No-op PROPPATCH handler for Apple Calendar setup compatibility |
| Auth middleware | `server/internal/caldav/auth.go` | HTTP Basic auth for CalDAV clients; validate against user store |
| Calendar revision token | `server/internal/caldav/sync.go` | Maintain `calendars.sync_token` as an internal revision token; optionally expose `DAV:sync-token` property; no RFC 6578 sync-collection REPORT support in MVP |
| Integration tests | `server/internal/caldav/handler_test.go` | httptest-based CalDAV operation tests (PROPFIND, PUT, GET, DELETE) |

**Key Implementation Details:**

1. **CalDAV Backend Adapter:**
   ```go
   type Backend struct {
       eventRepo EventRepo
       calRepo   CalendarRepo
   }

   func (b *Backend) Calendar(ctx context.Context, path string) (*caldav.Calendar, error) {
       // Query calendars table, return go-webdav Calendar struct
   }

   func (b *Backend) GetCalendarObject(ctx context.Context, path string, req *caldav.CalendarCompQuery) (*caldav.CalendarObject, error) {
       // Parse path to extract UID, query events table by UID
       // Return CalendarObject with Data = ICS string, ETag = events.etag
   }

	   func (b *Backend) PutCalendarObject(ctx context.Context, path string, cal *ical.Calendar, opts *caldav.PutCalendarObjectOptions) (string, error) {
	       // Extract UID from iCalendar
	       // Generate ETag = SHA256(icsBytesPersisted)
	       // Check If-Match header (opts.IfMatch) against stored ETag
	       // If mismatch, return caldav.ErrPreconditionFailed (maps to HTTP 412)
	       // Otherwise, save ICS + extract metadata, store in events table
	       // Return new ETag
	   }
   ```

2. **ETag Strategy:**
   - **Generation:** `ETag = hex(sha256(icsBytes))`
   - **Storage:** `events.etag` column (updated on every write)
   - **Validation:** On PUT with If-Match header, compare client ETag vs. stored; reject if mismatch (HTTP 412)
   - **Purpose:** Enables conflict detection for concurrent edits from multiple clients

3. **Sync-Token Strategy:**
   - **Storage:** `calendars.sync_token` column
   - **Generation:** Increment on any calendar change (event create/update/delete)
   - **Format:** Simple integer or timestamp-based token
   - **Usage (MVP):** Internal revision token for SSE/UI caching and future incremental sync support (do not rely on CalDAV clients requesting changes since token)
   - **MVP:** No `DAV:sync-collection` REPORT support; clients sync via standard list/get + ETags

4. **PROPPATCH No-Op:**
   - Apple Calendar issues PROPPATCH on account setup
   - Handler returns HTTP 207 Multi-Status with all properties marked as "failed" or "not supported"
   - Prevents setup failure; clients proceed with core operations

5. **CalDAV Mounting:**
   - Mount at `/dav` (configured in Vite proxy, main.go)
   - Structure: `/dav/calendars/{username}/{calendar-name}/`
   - Principal URL: `/dav/principals/{username}/`
   - Well-known URLs: `/.well-known/caldav` → `/dav/`

**Testing Plan:**

**Unit Tests:**
```go
// server/internal/caldav/backend_test.go
func TestBackend_PutCalendarObject_ETagMismatch(t *testing.T) {
    // Create event with ETag "abc123"
    // Attempt PUT with If-Match: "wrong-etag"
    // Expect: caldav.ErrPreconditionFailed (HTTP 412)
}

func TestBackend_PutCalendarObject_Success(t *testing.T) {
    // PUT new event, verify stored in DB with correct ETag
    // GET same event, verify ETag matches
}
```

**Integration Tests (httptest):**
```go
// server/internal/caldav/handler_test.go
func TestCalDAV_PROPFIND_ListEvents(t *testing.T) {
    // Seed DB with 2 events
    // Send PROPFIND request with Depth: 1
    // Verify response includes both events with correct ETags
}

func TestCalDAV_PUT_GET_Roundtrip(t *testing.T) {
    // PUT iCalendar event via CalDAV
    // GET same event
    // Verify ICS matches exactly
}

func TestCalDAV_ConcurrentEdit_Conflict(t *testing.T) {
    // Client A gets event with ETag "v1"
    // Client B updates event (new ETag "v2")
    // Client A tries PUT with If-Match: "v1"
    // Expect: HTTP 412 Precondition Failed
}
```

**Manual Interop Checklist:**

After Phase B implementation, perform these tests:

| Client | Test | Expected Result |
|--------|------|-----------------|
| **Apple Calendar (macOS)** | Add CalDAV account at `http://localhost:8080/dav/` | Account setup succeeds (PROPPATCH no-op works) |
| | Create event in Apple Calendar | Event appears in Calendar-app DB; `events` table has new row with ICS + ETag |
| | Edit event in Apple Calendar | Event updates in DB; ETag changes |
| **Fantastical** | Add CalDAV account | Setup succeeds; calendar list appears |
| | Create recurring event (weekly) | Event stored with RRULE; appears in other clients |
| **DAVx⁵ (Android)** | Configure CalDAV account | Sync succeeds; no duplicate events |
| | Delete event from Android | Event deleted in DB; disappears from other clients |
| **Conflict Test** | Edit same event from two clients simultaneously | Second edit returns error; client prompts for conflict resolution |

**Validation Commands:**

```bash
# Server compiles and runs
cd server && go build ./cmd/calendarapp
./calendar-app  # Should start on :8080

# Health check responds
curl http://localhost:8080/health  # Returns {"status":"ok"}

# CalDAV discovery works
curl -X PROPFIND http://localhost:8080/.well-known/caldav  # Should redirect or return principal

# Database initialized
ls data/calendar.db  # File exists
sqlite3 data/calendar.db "SELECT name FROM sqlite_master WHERE type='table';"
# Should list: users, calendars, events, tasks, preferences, audit_log

# All tests pass
cd server && go test ./...
```

**Acceptance Criteria:**
- [ ] CalDAV PROPFIND returns event list correctly
- [ ] CalDAV PUT creates event; GET retrieves identical ICS
- [ ] ETag conflict detection works (HTTP 412 on mismatch)
- [ ] At least 2 of 3 clients (Apple Calendar, Fantastical, DAVx⁵) sync successfully in manual testing
- [ ] Zero data loss in roundtrip tests (PUT → GET → compare)
- [ ] Integration tests pass via `go test ./internal/caldav/...`

---

### Phase C: Constraint Validation Engine + Planning API Stub

**Goal:** Implement deterministic constraint checker that gates all AI-proposed plans, and create `/api/v1/plan/daily` stub endpoint.

**Deliverables:**

| Component | File Path | Responsibilities |
|-----------|-----------|------------------|
| Constraint engine | `server/internal/planner/constraints.go` | Deterministic validation: check for overlaps, time window violations, hard conflicts |
| Constraint types | `server/internal/planner/types.go` | Define `Constraint`, `Violation`, `ValidationResult` structs |
| Planning API | `server/internal/api/plan.go` | `/api/v1/plan/daily` endpoint returning mock proposal |
| Proposal types | `server/internal/api/types.go` | `PlanProposal`, `ProposedChange` structs matching UX diff format |
| Constraint tests | `server/internal/planner/constraints_test.go` | Unit tests for all constraint scenarios |

**Key Implementation Details:**

1. **Constraint Validation Engine:**

   **Purpose:** Gate all actionable content; nothing shown or applied without passing validation.

   **Constraint Types (MVP):**
   - **TimeConflict:** Event overlaps with existing event
   - **TimeWindowViolation:** Event outside allowed time window (e.g., not during work hours)
   - **ImmovableConflict:** Proposed change conflicts with user-tagged "immovable" event

   **Interface:**
   ```go
   type Constraint interface {
       Validate(ctx context.Context, proposal *Plan, existing []*Event) (*ValidationResult, error)
   }

   type ValidationResult struct {
       Valid      bool
       Violations []Violation
   }

   type Violation struct {
       Type        string  // "time_conflict", "immovable_conflict", etc.
       Message     string
       ConflictingEventID string  // Reference to blocking event
   }

   func ValidatePlan(ctx context.Context, proposal *Plan, constraints []Constraint, existingEvents []*Event) (*ValidationResult, error) {
       // Run all constraints
       // If any violation, result.Valid = false
       // Return violations for UI to display
   }
   ```

2. **Planning API Stub:**

   **Endpoint:** `POST /api/v1/plan/daily`

   **Request:**
   ```json
   {
     "date": "2026-01-16",
     "todoist_tasks": [],  // Empty for now (Todoist integration Phase D)
     "preferences": {}
   }
   ```

   **Response (Mock Data):**
   ```json
   {
     "proposal_id": "prop-12345",
     "date": "2026-01-16",
     "changes": [
       {
         "type": "add",
         "entity": "focus_block",
         "summary": "Deep Work - Project Planning",
         "start_time": "2026-01-16T09:00:00Z",
         "end_time": "2026-01-16T11:00:00Z",
         "reasoning": "Available 2-hour slot identified; matches peak energy preference"
       }
     ],
     "validation_result": {
       "valid": true,
       "violations": []
     }
   }
   ```

3. **Constraint Gate Integration:**

   **Flow:**
   ```
   1. Generate mock proposal (Phase C stub; later: real LLM)
   2. Run constraint validation engine
   3. If violations found, either:
      a) Filter out violating changes
      b) Return proposal with valid=false + violations for UI
   4. Return only validated proposals to client
   ```

   **Critical:** No proposal ever reaches the UI without passing `ValidatePlan()`.

4. **Apply Endpoint (Stub):**

   **Endpoint:** `POST /api/v1/plan/apply`

   **Request:**
   ```json
   {
     "proposal_id": "prop-12345"
   }
   ```

   **Response:**
   ```json
   {
     "applied": true,
     "changes_made": 3,
     "rollback_window_expires_at": "2026-01-16T10:15:00Z"
   }
   ```

   **Implementation (Phase C):**
   - Look up proposal by ID (in-memory cache or DB)
   - Re-validate constraints (in case state changed)
   - If still valid, create events via EventRepo
   - Record to audit_log for rollback
   - Return success

**Testing Plan:**

**Unit Tests:**
```go
// server/internal/planner/constraints_test.go
func TestTimeConflictConstraint_Overlap(t *testing.T) {
    // Existing event: 9am-10am
    // Proposed event: 9:30am-10:30am
    // Expect: Violation with type="time_conflict"
}

func TestValidatePlan_AllConstraintsPass(t *testing.T) {
    // Proposal with no conflicts
    // Expect: Valid=true, Violations=empty
}
```

**Integration Tests:**
```go
// server/internal/api/plan_test.go
func TestPlanDaily_ReturnsValidatedProposal(t *testing.T) {
    // Seed calendar with event at 2pm-3pm
    // Request daily plan
    // Verify: proposed events don't overlap with 2pm-3pm
}

func TestPlanApply_CreatesEventsInCalendar(t *testing.T) {
    // Generate proposal
    // Apply via API
    // Verify: events exist in DB
    // Verify: audit_log entry created
}
```

**Validation Commands:**

```bash
# Constraint tests pass
cd server && go test ./internal/planner/...

# Planning API stub responds
curl -X POST http://localhost:8080/api/v1/plan/daily \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-01-16"}'
# Should return mock proposal JSON

# Validation integration works
# (Seed DB with conflicting event, request plan, verify proposal avoids it)
```

**Acceptance Criteria:**
- [ ] Constraint validation engine exists with at least 2 constraint types (TimeConflict, TimeWindowViolation)
- [ ] `ValidatePlan()` returns violations for invalid proposals
- [ ] `/api/v1/plan/daily` returns mock proposal with validation_result
- [ ] `/api/v1/plan/apply` creates events in calendar (via EventRepo)
- [ ] Audit log records apply actions with rollback data
- [ ] Unit tests pass for all constraint scenarios
- [ ] Integration test: plan API doesn't propose overlapping events

---

## 4. Interfaces / Data Model Notes

### Event Storage Strategy

**Hybrid Approach:** Store both ICS and extracted metadata

**Rationale:**
- **ICS string:** Preserves full iCalendar fidelity for CalDAV clients (no data loss from parsing/re-encoding)
- **Extracted metadata:** Enables efficient SQL queries for planning logic (find events in time range, check conflicts)

**Schema (already in `00001_init.sql`):**
```sql
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    calendar_id INTEGER NOT NULL,
    uid TEXT NOT NULL,           -- iCalendar UID
    ics TEXT NOT NULL,           -- Full VEVENT component
    summary TEXT,                -- Extracted for queries
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    all_day BOOLEAN DEFAULT 0,
    recurrence_rule TEXT,        -- RRULE if recurring
    etag TEXT NOT NULL,          -- SHA-256(ics)
    sequence INTEGER DEFAULT 0,
    status TEXT DEFAULT 'CONFIRMED',
    ...
    UNIQUE(calendar_id, uid)
);
```

**Write Flow:**
1. Client sends iCalendar via CalDAV PUT
2. Parse ICS to extract UID, summary, start_time, end_time, RRULE
3. Generate ETag = `sha256(icsBytes)`
4. Store ICS (as-is) + extracted metadata
5. Return ETag in response header

**Read Flow:**
1. Client requests event via CalDAV GET or REPORT
2. Query DB by UID or time range
3. Return `events.ics` directly (no re-encoding)
4. Include ETag in response header

### ETag Generation

**Algorithm:**
```go
import "crypto/sha256"

func GenerateETag(icsData []byte) string {
    hash := sha256.Sum256(icsData)
    return fmt.Sprintf(`"%x"`, hash)  // Quoted string per HTTP spec
}
```

**Properties:**
- Deterministic: Same ICS → same ETag
- Collision-resistant: Different ICS → different ETag (with SHA-256 probability)
- Fast: Hashing is O(n) on ICS size

### Calendar Revision Token (`sync_token`)

**MVP Approach:** Internal calendar-level version counter (useful for SSE/UI caching and future CalDAV incremental sync support)

**Schema:**
```sql
-- Already in calendars table:
sync_token TEXT
```

**Implementation:**
```go
// On any event mutation (create/update/delete):
func (r *SQLiteCalendarRepo) IncrementSyncToken(ctx context.Context, calendarID int64) (string, error) {
    // Option 1: Timestamp-based
    token := fmt.Sprintf("v%d", time.Now().Unix())

    // Option 2: Counter-based
    // Query current sync_token, parse version number, increment

    // UPDATE calendars SET sync_token = ? WHERE id = ?
    return token, nil
}
```

**WebDAV sync-collection (RFC 6578):**
- Deferred to post-MVP per scope decisions (no `DAV:sync-collection` REPORT in MVP)
- MVP uses standard CalDAV list/get + ETags (PROPFIND/REPORT per implemented subset)
- `calendars.sync_token` is maintained as an internal revision token; do not depend on client incremental sync in MVP

### Constraint Validation Data Structures

**Proposal Representation:**
```go
type Plan struct {
    ID        string
    Date      time.Time
    Changes   []ProposedChange
}

type ProposedChange struct {
    Type      string  // "add", "move", "delete"
    Event     *Event  // Proposed event details
    Reasoning string
}
```

**Constraint Interface:**
```go
type Constraint interface {
    Validate(ctx context.Context, proposal *Plan, existingEvents []*Event) (*ValidationResult, error)
}
```

**Built-in Constraints (Phase C):**

1. **TimeConflictConstraint:** No overlaps with existing confirmed events
2. **TimeWindowConstraint:** Events must be within allowed hours (e.g., 9am-5pm)
3. **ImmovableConstraint:** Don't schedule over events tagged as immovable

**Validation Flow:**
```
Proposal → ValidatePlan(proposal, constraints, existingEvents)
          → If violations found: filter or reject
          → Only valid proposals reach UI
```

---

## 5. Testing Plan

### Unit Testing Strategy

**Target Coverage:** Core business logic, not glue code

| Package | Test Files | Focus |
|---------|-----------|-------|
| `internal/domain` | `event_test.go`, `calendar_test.go` | Domain model validation methods |
| `internal/data` | `sqlite_event_repo_test.go` | Repository CRUD operations, ETag handling |
| `internal/caldav` | `backend_test.go` | CalDAV Backend adapter logic |
| `internal/planner` | `constraints_test.go` | All constraint scenarios (overlap, time window, immovable) |
| `internal/api` | `plan_test.go` | API endpoint request/response handling |

**Test Pattern (Table-Driven):**
```go
func TestTimeConflictConstraint(t *testing.T) {
    tests := []struct {
        name      string
        existing  []*Event
        proposed  *Event
        wantValid bool
    }{
        {"no_overlap", []*Event{{Start: 9am, End: 10am}}, &Event{Start: 10am, End: 11am}, true},
        {"overlap", []*Event{{Start: 9am, End: 10am}}, &Event{Start: 9:30am, End: 10:30am}, false},
        // ... more cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Run validation
            // Assert result.Valid == tt.wantValid
        })
    }
}
```

### Integration Testing Strategy

**httptest-based CalDAV Tests:**

```go
// server/internal/caldav/handler_test.go
func TestCalDAVIntegration(t *testing.T) {
    // 1. Initialize in-memory SQLite DB
    // 2. Run migrations
    // 3. Create CalDAV handler
    // 4. Use httptest.Server
    // 5. Send CalDAV requests (PROPFIND, PUT, GET, DELETE)
    // 6. Verify DB state and response correctness
}
```

**Example Test:**
```go
func TestPutGetRoundtrip(t *testing.T) {
    srv := setupTestServer(t)
    defer srv.Close()

    // PUT event
    icsData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-event-1
SUMMARY:Test Event
DTSTART:20260116T090000Z
DTEND:20260116T100000Z
END:VEVENT
END:VCALENDAR`

    req := httptest.NewRequest("PUT", "/dav/calendars/testuser/cal1/test-event-1.ics", strings.NewReader(icsData))
    req.Header.Set("Content-Type", "text/calendar")

    resp := executeRequest(srv, req)
    assert.Equal(t, http.StatusCreated, resp.Code)
    etag := resp.Header().Get("ETag")
    assert.NotEmpty(t, etag)

    // GET event
    req = httptest.NewRequest("GET", "/dav/calendars/testuser/cal1/test-event-1.ics", nil)
    resp = executeRequest(srv, req)
    assert.Equal(t, http.StatusOK, resp.Code)
    assert.Equal(t, etag, resp.Header().Get("ETag"))

    // Verify ICS matches
    body := resp.Body.String()
    assert.Contains(t, body, "UID:test-event-1")
    assert.Contains(t, body, "SUMMARY:Test Event")
}
```

### Manual Interop Testing (Critical)

**After Phase B, perform manual verification:**

**Checklist:**

1. **Apple Calendar (macOS) Setup**
   - [ ] Open Apple Calendar → Add Account → Other CalDAV Account
   - [ ] Enter: Server: `http://localhost:8080/dav/`, Username: `testuser`, Password: `testpass`
   - [ ] Verify: Account setup succeeds without errors
   - [ ] Create event in Apple Calendar
   - [ ] Verify: Event appears in `data/calendar.db` (query `events` table)

2. **Fantastical Setup**
   - [ ] Add CalDAV account with same credentials
   - [ ] Verify: Existing events from Apple Calendar appear
   - [ ] Create recurring event (weekly meeting)
   - [ ] Verify: Event has RRULE in DB, appears in Apple Calendar

3. **DAVx⁵ Setup (Android)**
   - [ ] Install DAVx⁵ on Android device or emulator
   - [ ] Add CalDAV account pointing to `http://<local-ip>:8080/dav/`
   - [ ] Verify: No duplicate events
   - [ ] Delete event from DAVx⁵
   - [ ] Verify: Event deleted from DB, disappears from desktop clients

4. **Conflict Resolution Test**
   - [ ] Open same event in two clients (e.g., Apple Calendar on Mac + iPhone)
   - [ ] Edit in both simultaneously (change title, time)
   - [ ] Save on Client A, then Client B
   - [ ] Verify: Client B receives conflict error (sync shows conflict or refresh reveals discrepancy)
   - [ ] Verify: No data corruption; one version preserved

**Document Results:**
- Create `docs/testing/caldav-interop-results.md` with screenshots and logs
- Note any client-specific quirks or workarounds needed

---

## 6. Acceptance Criteria

### Phase A: Domain + Repository

**Definition of Done:**

- [ ] **Domain models exist** with validation methods
  ```bash
  # Check files exist
  ls server/internal/domain/event.go
  ls server/internal/domain/repository.go
  ```

- [ ] **Repository interfaces defined** (EventRepo, CalendarRepo)
  ```bash
  # Check interfaces compile
  cd server && go build ./internal/domain/...
  ```

- [ ] **SQLite implementation complete**
  ```bash
  # Check implementation exists
  ls server/internal/data/sqlite_event_repo.go

  # Unit tests pass
  cd server && go test ./internal/data/... -v
  ```

- [ ] **Migration runner works**
  ```bash
  # Clean data dir
  rm -rf data/

  # Run with migration flag
  cd server && go run ./cmd/calendarapp --migrate-only

  # Verify schema
  sqlite3 data/calendar.db ".schema events" | grep -q "etag TEXT NOT NULL"
  ```

- [ ] **ETag generation tested**
  ```bash
  # Test in unit tests; verify same ICS → same ETag
  cd server && go test ./internal/data/... -run TestETagGeneration
  ```

### Phase B: CalDAV CRUD + Sync

**Definition of Done:**

- [ ] **CalDAV handler mounted at `/dav`**
  ```bash
  # Server starts without error
  cd server && go run ./cmd/calendarapp &
  SERVER_PID=$!

  # Well-known endpoint works
  curl -i http://localhost:8080/.well-known/caldav
  # Should return 301 or principal URL

  # Kill server
  kill $SERVER_PID
  ```

- [ ] **PROPFIND returns event list**
  ```bash
  # Seed DB with test event
  # Send PROPFIND request
  curl -X PROPFIND http://localhost:8080/dav/calendars/testuser/cal1/ \
    -H "Depth: 1" \
    -H "Content-Type: application/xml" \
    --data '<propfind xmlns="DAV:"><prop><getetag/></prop></propfind>'
  # Should return XML with events
  ```

- [ ] **PUT/GET roundtrip preserves ICS**
  ```bash
  # Run integration test
  cd server && go test ./internal/caldav/... -run TestPutGetRoundtrip
  # Must pass
  ```

- [ ] **ETag conflict detection works**
  ```bash
  cd server && go test ./internal/caldav/... -run TestETagConflict
  # Must return HTTP 412 on mismatch
  ```

- [ ] **Manual client interop succeeds** (at least 2 of 3)
  - Apple Calendar: Setup completes, events sync
  - Fantastical: Setup completes, events sync
  - DAVx⁵: Setup completes, events sync

  **Evidence:** Screenshots or logs showing successful sync

### Phase C: Constraint Validation + Planning Stub

**Definition of Done:**

- [ ] **Constraint engine exists**
  ```bash
  ls server/internal/planner/constraints.go
  ls server/internal/planner/types.go
  ```

- [ ] **Constraint tests pass**
  ```bash
  cd server && go test ./internal/planner/... -v
  # All scenarios covered: overlap, time window, immovable conflicts
  ```

- [ ] **Planning API responds**
  ```bash
  # Start server
  cd server && go run ./cmd/calendarapp &

  # Request daily plan
  curl -X POST http://localhost:8080/api/v1/plan/daily \
    -H "Content-Type: application/json" \
    -d '{"date":"2026-01-16"}' | jq .

  # Should return proposal JSON with validation_result.valid=true

  # Kill server
  kill $!
  ```

- [ ] **Apply endpoint creates events**
  ```bash
  # Integration test verifies
  cd server && go test ./internal/api/... -run TestPlanApply
  ```

- [ ] **Audit log records applies**
  ```bash
  # After apply, check audit_log table
  sqlite3 data/calendar.db "SELECT action, entity_type FROM audit_log WHERE action='plan_apply';"
  # Should show entry
  ```

- [ ] **Invalid proposals rejected**
  ```bash
  # Seed DB with event at 2pm-3pm
  # Manually create proposal with overlapping event
  # Run validation
  # Verify: validation_result.valid=false with violation details
  ```

---

## 7. Deliverables

### Phase A Deliverable

**PR Title:** `feat(server): implement domain model + repository layer`

**Files Changed:**
```
server/internal/domain/event.go
server/internal/domain/calendar.go
server/internal/domain/task.go
server/internal/domain/repository.go
server/internal/data/db.go
server/internal/data/migrations.go
server/internal/data/sqlite_event_repo.go
server/internal/data/sqlite_calendar_repo.go
server/internal/data/sqlite_event_repo_test.go
server/cmd/calendarapp/main.go  (wire DB connection + migrations)
```

**Tests Required:**
- [ ] `go test ./internal/domain/...` passes
- [ ] `go test ./internal/data/...` passes
- [ ] Migration runner tested with `--migrate-only` flag

**Docs Updates:**
- Add `docs/architecture/data-layer.md` explaining repository pattern and ETag strategy

---

### Phase B Deliverable

**PR Title:** `feat(server): implement CalDAV CRUD with multi-client sync`

**Files Changed:**
```
server/internal/caldav/backend.go
server/internal/caldav/handler.go
server/internal/caldav/proppatch.go
server/internal/caldav/auth.go
server/internal/caldav/sync.go
server/internal/caldav/backend_test.go
server/internal/caldav/handler_test.go
server/cmd/calendarapp/main.go  (mount CalDAV handler at /dav)
docs/testing/caldav-interop-results.md  (manual test results)
```

**Tests Required:**
- [ ] `go test ./internal/caldav/...` passes (integration tests with httptest)
- [ ] Manual interop checklist completed (2+ clients working)
- [ ] Zero data loss verified (roundtrip tests pass)

**Docs Updates:**
- Add `docs/testing/caldav-interop-results.md` with manual test results
- Update README.md with CalDAV setup instructions for clients

**Validation Evidence:**
- Screenshots of Apple Calendar setup succeeding
- Logs showing successful PROPFIND/PUT/GET operations
- DB dump showing events with correct ETags

---

### Phase C Deliverable

**PR Title:** `feat(server): add constraint validation engine + planning API stub`

**Files Changed:**
```
server/internal/planner/constraints.go
server/internal/planner/types.go
server/internal/planner/time_conflict.go
server/internal/planner/constraints_test.go
server/internal/api/plan.go
server/internal/api/types.go
server/internal/api/plan_test.go
server/cmd/calendarapp/main.go  (mount /api/v1/plan/* routes)
```

**Tests Required:**
- [ ] `go test ./internal/planner/...` passes (all constraint scenarios)
- [ ] `go test ./internal/api/...` passes (planning endpoint tests)
- [ ] Integration test: invalid proposals rejected

**Docs Updates:**
- Add `docs/architecture/constraint-validation.md` explaining how constraints work
- Update README.md with API endpoint documentation

**Validation Evidence:**
- Constraint tests covering 5+ scenarios
- API returning mock proposals with validation results
- Proof that overlapping proposals are rejected

---

## 8. Critical Implementation Guidance

### CalDAV Protocol Gotchas

Based on research and `emersion/go-webdav` usage:

1. **PROPPATCH No-Op:**
   - Apple Calendar **requires** PROPPATCH response on setup
   - Return HTTP 207 Multi-Status with properties marked `<status>HTTP/1.1 403 Forbidden</status>`
   - This satisfies client without implementing full property editing

2. **Well-Known URLs:**
   - Implement `/.well-known/caldav` → redirect to principal URL
   - Clients use this for auto-discovery

3. **Principal Discovery:**
   - Return `calendar-home-set` property pointing to `/dav/calendars/{username}/`
   - Required for clients to find calendar collections

4. **ETag Quoting:**
   - HTTP spec requires ETags to be quoted strings: `ETag: "abc123"`
   - Don't forget quotes or clients may not recognize

5. **Sync Reliability:**
   - Always return the same ICS bytes the client sent (don't re-encode)
   - This prevents subtle formatting differences that break client sync

### Constraint Validation Critical Rules

1. **Deterministic Only:**
   - No probabilistic checks
   - No ML-based validation
   - If same proposal + same calendar state → same validation result

2. **Fail-Safe:**
   - On validation error (e.g., can't query DB), reject proposal
   - Never allow an unvalidated proposal through

3. **Clear Violation Messages:**
   - `{"type": "time_conflict", "message": "Overlaps with 'CEO Meeting' at 2pm", "conflicting_event_id": "evt-123"}`
   - UI needs this to explain why proposal was rejected

### Error Handling Standards

**CalDAV Errors:**
- HTTP 412 Precondition Failed: ETag mismatch
- HTTP 409 Conflict: UID collision
- HTTP 404 Not Found: Event doesn't exist
- HTTP 403 Forbidden: Not authorized

**REST API Errors:**
```json
{
  "error": "constraint_violation",
  "message": "Proposed plan overlaps with immovable CEO meeting",
  "details": {
    "conflicting_event_id": "evt-456",
    "violation_type": "immovable_conflict"
  }
}
```

**Logging on Errors:**
```go
slog.Error("CalDAV PUT failed",
    "path", path,
    "error", err,
    "user_id", userID,
    "client_user_agent", r.Header.Get("User-Agent"),
)
```

---

## 9. Dependencies & Prerequisites

### External Libraries to Install

**Go (server/):**
```bash
go get github.com/emersion/go-webdav@latest
go get github.com/go-chi/chi/v5@latest
go get github.com/glebarez/sqlite@latest
go get github.com/pressly/goose/v3@latest
go get github.com/emersion/go-ical@latest  # For parsing iCalendar
```

**React (web/):**
```bash
pnpm install @tanstack/react-query react-router-dom zustand react-hook-form zod
pnpm install -D tailwindcss postcss autoprefixer
```

### Development Environment

**Required:**
- Go 1.22+
- Node.js 18+ with pnpm
- SQLite3 CLI (for manual DB inspection)
- curl (for API testing)

**Optional but recommended:**
- golangci-lint (for linting)
- reflex or air (for auto-reload)
- Apple Calendar or Fantastical (for interop testing)

---

## 10. Risk Mitigation

### Top Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **CalDAV client incompatibility** | Medium | High | Implement PROPPATCH no-op; test with 3 clients early; use Litmus suite |
| **ETag conflict handling bugs** | Medium | High | Table-driven tests for all conflict scenarios; integration tests with concurrent requests |
| **SQLite concurrent access issues** | Low | Medium | WAL mode enabled; short transactions; load test with multiple clients |
| **Constraint validation false positives** | Medium | Medium | Comprehensive test suite; log all rejections for debugging |
| **Data loss on sync** | Low | Critical | Roundtrip tests (PUT → GET → compare); never re-encode ICS; use client's bytes |

### Validation Checkpoints

**After Phase A:**
- Can create event via repository → query it back with correct ETag
- Migration creates all tables with correct indexes

**After Phase B:**
- At least one CalDAV client (preferably Apple Calendar) syncs successfully
- Roundtrip test passes (client's ICS bytes returned unchanged)
- Conflict test triggers HTTP 412 correctly

**After Phase C:**
- Constraint validation rejects overlapping events
- Planning API returns validated proposals
- Apply endpoint creates events + audit log entry

**Go/No-Go Decision Points:**

- **After Phase A:** If repo layer tests fail, don't proceed to CalDAV
- **After Phase B:** If 0 of 3 clients sync correctly, investigate before Phase C
- **After Phase C:** If constraint validation has false positives, fix before continuing

---

## 11. Success Metrics

### Quantitative Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| CalDAV operation latency | <500ms | httptest timing; manual client sync observation |
| ETag collision rate | 0% | 1000 events → 1000 unique ETags (unit test) |
| Concurrent PUT conflicts detected | 100% | Integration test with race condition |
| Test coverage (data layer) | >80% | `go test -cover ./internal/data/...` |
| Manual interop success | 2+ clients | Apple Calendar + Fantastical or DAVx⁵ working |

### Qualitative Criteria

- **Code clarity:** Repository interfaces are self-documenting; no magic
- **Error messages:** CalDAV errors include enough context to debug (user_id, path, error reason)
- **Testability:** Can test constraint logic without spinning up full server
- **Logs:** Structured logging with clear event context (CalDAV operation, user, timestamp)

---

## 12. Implementation Notes

### Recommended Implementation Order

**Phase A:**
1. Define domain models (`domain/event.go`, `domain/calendar.go`)
2. Define repository interfaces (`domain/repository.go`)
3. Implement SQLite repos (`data/sqlite_event_repo.go`)
4. Wire DB connection + migrations in `main.go`
5. Write unit tests for repo operations
6. Verify: `go test ./internal/data/...` passes

**Phase B:**
1. Implement `caldav.Backend` interface (`caldav/backend.go`)
2. Create CalDAV handler with auth (`caldav/handler.go`, `caldav/auth.go`)
3. Implement PROPPATCH no-op (`caldav/proppatch.go`)
4. Mount handler at `/dav` in `main.go`
5. Write integration tests (httptest)
6. Manual testing with Apple Calendar
7. Fix any client-specific issues
8. Test with Fantastical and DAVx⁵
9. Document interop results

**Phase C:**
1. Define constraint interfaces (`planner/types.go`)
2. Implement TimeConflictConstraint (`planner/time_conflict.go`)
3. Implement ValidatePlan function (`planner/constraints.go`)
4. Write constraint unit tests
5. Create planning API types (`api/types.go`)
6. Implement `/api/v1/plan/daily` stub (`api/plan.go`)
7. Integrate constraint validation into planning endpoint
8. Implement `/api/v1/plan/apply` stub
9. Add audit logging on apply
10. Write API integration tests

### Code Style Guidelines

**Go:**
- Use `slog` for all logging (no fmt.Println)
- Handle all errors explicitly (no `_` discards unless justified)
- Use context.Context for cancellation
- Keep functions small (<50 lines)
- Table-driven tests for multiple scenarios

**Error Handling:**
```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create event %s: %w", uid, err)
}

// Good: Log before returning errors
slog.Error("CalDAV PUT failed", "uid", uid, "error", err)
return err
```

**Transaction Pattern:**
```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()  // No-op if Commit succeeds

// ... do work with tx

return tx.Commit()
```

---

## 13. Reference Implementation Patterns

### CalDAV Backend Adapter Example

```go
// server/internal/caldav/backend.go
package caldav

import (
    "context"
    "github.com/emersion/go-webdav/caldav"
    "github.com/yourorg/calendar-app/server/internal/domain"
)

type Backend struct {
    eventRepo    domain.EventRepo
    calendarRepo domain.CalendarRepo
}

func NewBackend(eventRepo domain.EventRepo, calendarRepo domain.CalendarRepo) *Backend {
    return &Backend{
        eventRepo:    eventRepo,
        calendarRepo: calendarRepo,
    }
}

func (b *Backend) GetCalendarObject(ctx context.Context, path string, req *caldav.CalendarCompQuery) (*caldav.CalendarObject, error) {
    // Parse path to extract calendar ID and UID
    calID, uid, err := parsePath(path)
    if err != nil {
        return nil, err
    }

    // Query event from repository
    event, err := b.eventRepo.GetByUID(ctx, calID, uid)
    if err != nil {
        return nil, err
    }

    // Return CalDAV object with ICS and ETag
    return &caldav.CalendarObject{
        Path:         path,
        ETag:         event.ETag,
        ModTime:      event.LastModified,
        ContentType:  "text/calendar",
        Data:         strings.NewReader(event.ICS),
    }, nil
}

func (b *Backend) PutCalendarObject(ctx context.Context, path string, cal *ical.Calendar, opts *caldav.PutCalendarObjectOptions) (string, error) {
    // Extract UID from iCalendar
    uid := extractUID(cal)

    // Encode iCalendar to bytes
    var buf bytes.Buffer
    if err := cal.EncodeTo(&buf); err != nil {
        return "", err
    }
    icsBytes := buf.Bytes()

    // Generate ETag
    etag := generateETag(icsBytes)

    // Check If-Match precondition
    if opts.IfMatch != "" && opts.IfMatch != etag {
        // ETag mismatch → conflict
        return "", caldav.ErrPreconditionFailed
    }

    // Extract metadata for SQL queries
    summary, startTime, endTime, rrule := extractMetadata(cal)

    // Save to repository
    event := &domain.Event{
        CalendarID:     calID,
        UID:            uid,
        ICS:            string(icsBytes),
        Summary:        summary,
        StartTime:      startTime,
        EndTime:        endTime,
        RecurrenceRule: rrule,
        ETag:           etag,
        Sequence:       extractSequence(cal),
    }

    if err := b.eventRepo.Create(ctx, event); err != nil {
        return "", err
    }

    return etag, nil
}
```

### Constraint Validation Example

```go
// server/internal/planner/time_conflict.go
package planner

type TimeConflictConstraint struct{}

func (c *TimeConflictConstraint) Validate(ctx context.Context, proposal *Plan, existingEvents []*domain.Event) (*ValidationResult, error) {
    violations := []Violation{}

    for _, change := range proposal.Changes {
        if change.Type != "add" && change.Type != "move" {
            continue  // Only check additions/moves
        }

        proposedEvent := change.Event

        // Check against all existing events
        for _, existing := range existingEvents {
            if eventsOverlap(proposedEvent, existing) {
                violations = append(violations, Violation{
                    Type:               "time_conflict",
                    Message:            fmt.Sprintf("Overlaps with '%s' at %s", existing.Summary, existing.StartTime.Format("3:04pm")),
                    ConflictingEventID: existing.UID,
                })
            }
        }
    }

    return &ValidationResult{
        Valid:      len(violations) == 0,
        Violations: violations,
    }, nil
}

func eventsOverlap(e1, e2 *domain.Event) bool {
    // Returns true if time ranges overlap
    return e1.StartTime.Before(e2.EndTime) && e2.StartTime.Before(e1.EndTime)
}
```

---

## 14. Glossary

| Term | Definition |
|------|------------|
| **ETag** | Entity Tag; HTTP header for cache validation and conflict detection; we use SHA-256(ICS) |
| **ICS** | iCalendar format; text representation of calendar data per RFC 5545 |
| **Sync-Token** | CalDAV concept; opaque string representing calendar version for incremental sync |
| **PROPFIND** | WebDAV/CalDAV method to query properties and list resources |
| **REPORT** | CalDAV method for complex queries (e.g., events in date range) |
| **PROPPATCH** | WebDAV method to modify properties; we implement as no-op for client compatibility |
| **Principal** | CalDAV concept; represents a user with calendars |
| **Calendar Home** | Collection URL containing user's calendars |

---

## 15. Verification Checklist

Use this checklist to verify each phase before moving forward:

### Phase A Verification

- [ ] Run: `cd server && go test ./internal/domain/...` → All pass
- [ ] Run: `cd server && go test ./internal/data/...` → All pass
- [ ] Run: `cd server && go run ./cmd/calendarapp --migrate-only` → DB created at `data/calendar.db`
- [ ] Run: `sqlite3 data/calendar.db ".tables"` → Shows: users, calendars, events, tasks, preferences, audit_log
- [ ] Run: `sqlite3 data/calendar.db "PRAGMA journal_mode;"` → Returns: `wal`
- [ ] Code review: Repository interfaces are clear and well-documented

### Phase B Verification

- [ ] Run: `cd server && go test ./internal/caldav/...` → All integration tests pass
- [ ] Run: `cd server && go run ./cmd/calendarapp` → Server starts without errors
- [ ] Run: `curl http://localhost:8080/health` → Returns `{"status":"ok"}`
- [ ] Run: `curl -X PROPFIND http://localhost:8080/.well-known/caldav` → Returns principal URL or redirects
- [ ] Manual: Add CalDAV account in Apple Calendar → Setup succeeds
- [ ] Manual: Create event in Apple Calendar → Event appears in DB
- [ ] Manual: Query DB: `sqlite3 data/calendar.db "SELECT uid, summary, etag FROM events;"` → Shows event with ETag
- [ ] Manual: Edit event in Apple Calendar → ETag changes in DB
- [ ] Manual: Connect second client (Fantastical or DAVx⁵) → Events sync

### Phase C Verification

- [ ] Run: `cd server && go test ./internal/planner/...` → All constraint tests pass
- [ ] Run: `cd server && go test ./internal/api/...` → Planning API tests pass
- [ ] Run: `curl -X POST http://localhost:8080/api/v1/plan/daily -H "Content-Type: application/json" -d '{"date":"2026-01-16"}'` → Returns validated proposal JSON
- [ ] Verify: Proposal includes `validation_result.valid = true`
- [ ] Manual: Seed DB with event at 2pm, request plan, verify proposal doesn't overlap
- [ ] Run: Apply stub → check `audit_log` table for entry
- [ ] Code review: Constraint validation is deterministic (no randomness, no external calls)

---

## 16. Post-Implementation: Return to Architecture Workflow

After all three phases are complete and verified:

1. **Update architecture.md** with implementation decisions from Phases A-C
2. **Resume Architecture workflow at Step 4** to document:
   - Data model details (as-built)
   - API surface design (endpoint specs)
   - Sync/conflict strategy (ETag implementation)
   - Failure modes & circuit breakers (Todoist/LLM graceful degradation)
   - Deployment/security/observability patterns
   - CalDAV interop test results

3. **Next development phases:**
   - Phase D: Todoist integration (read tasks, write scheduling metadata)
   - Phase E: Real AI planning (LLM harness, prompt engineering)
   - Phase F: Web UI implementation (calendar grid, proposal cards)

---

## 17. Questions for Clarification

If any of the following are unclear during implementation, refer back to source docs or ask:

1. **ETag format:** Is `"<sha256-hex>"` format (with quotes) correct per HTTP spec? → Yes, quoted strings required
2. **Sync-token persistence:** Should sync-token survive server restarts? → Yes, stored in `calendars.sync_token`
3. **Repository transactions:** Should Update() use optimistic locking? → Yes, check ETag before UPDATE
4. **Constraint ordering:** Does order of constraint checks matter? → No, all must pass; order doesn't affect result
5. **ICS normalization:** Should we normalize whitespace in ICS before hashing? → No, use client bytes exactly to avoid sync issues

---

## 18. Summary

This PRP defines a three-phase implementation sequence for Calendar-app MVP:

**Phase A:** Build the data layer with clean repository abstractions and ETag support.
**Phase B:** Implement CalDAV CRUD using emersion/go-webdav; validate multi-client sync with Apple Calendar, Fantastical, and DAVx⁵.
**Phase C:** Create deterministic constraint validation engine and planning API stub.

Each phase has clear acceptance criteria, testing requirements, and validation commands. The sequence prioritizes **CalDAV correctness first** (multi-client sync, zero data loss) before adding AI features, ensuring the foundation is solid.

**Success = All three phases verified + at least 2 CalDAV clients syncing reliably.**
