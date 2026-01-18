# PRP: GPT Pro Review — PR #4 Phase A+B CalDAV Implementation

**Type:** Code Review Prompt (GPT Pro Team)
**Repo:** `airplne/calendar-app`
**PR:** [#4](https://github.com/airplne/calendar-app/pull/4)
**Date:** 2026-01-18
**Related Implementation PRP:** `docs/prp/claude-dev-prp-mvp-sequence-caldav-repo-planner.md`

---

## 1) Context & Goal

Calendar-app is a self-hosted CalDAV server (source of truth) with a web UI and AI planning capabilities. We are executing the MVP sequence where **CalDAV multi-client sync correctness** must be proven before building AI features.

**PR #4 Goal:** Implement Phase A+B foundation:
- **Phase A:** Domain models + SQLite repository layer + tests
- **Phase B:** CalDAV CRUD endpoints + ETag conflict handling + client compatibility

### Locked Stack Decisions

| Layer | Technology | Notes |
|-------|------------|-------|
| **Backend** | Go 1.22+ | Single binary deployment |
| **Router** | `go-chi/chi/v5` | Custom method registration for WebDAV |
| **CalDAV** | `emersion/go-webdav` | CalDAV/WebDAV protocol implementation |
| **Database** | SQLite via `glebarez/sqlite` | Pure Go, no CGO |
| **Migrations** | `pressly/goose/v3` | SQL-based migrations |
| **Logging** | `log/slog` | Structured logging |
| **Frontend** | React + Vite + Tailwind | SPA with HashRouter |
| **Real-time** | SSE at `/events` | No WebSocket requirement |
| **CalDAV Mount** | `/dav` | Not `/calendars` |

### Why This PR Matters

1. CalDAV is the **source of truth** — all calendar data flows through it
2. Multi-client sync must work (Apple Calendar, Fantastical, DAVx5) before AI features
3. ETag-based conflict resolution prevents data loss
4. This foundation enables Phase C (AI planning) and beyond

---

## 2) What's Included in PR #4 (Expected Deliverables)

### Phase A: Domain/Repository Layer

| Deliverable | Expected Location | Purpose |
|-------------|-------------------|---------|
| Domain models | `server/internal/domain/` | Event, Calendar, Task, User structs |
| Repository interfaces | `server/internal/domain/repository.go` | Contracts for data access |
| SQLite repos | `server/internal/data/` | CRUD implementations |
| DB helpers | `server/internal/data/db.go` | Connection, migrations |
| Migrations | `server/migrations/00001_init.sql` | Schema definition |
| Repo tests | `server/internal/data/*_test.go` | Unit tests for repos |

**Schema Requirements:**
- `events` table must have `ics TEXT NOT NULL` (full ICS for roundtrip fidelity)
- `events` table must have `etag TEXT NOT NULL` (SHA-256 of persisted ICS)
- `calendars` table must have unique constraint on `(user_id, name)`
- Extracted metadata columns (summary, start_time, etc.) allowed but don't replace raw ICS

### Phase B: CalDAV Handler

| Deliverable | Expected Location | Purpose |
|-------------|-------------------|---------|
| CalDAV handler | `server/internal/caldav/handler.go` | Chi router setup |
| CalDAV backend | `server/internal/caldav/backend.go` | go-webdav backend impl |
| Auth middleware | `server/internal/caldav/auth.go` | HTTP Basic auth |
| PROPPATCH handler | `server/internal/caldav/proppatch.go` | Apple Calendar compat |
| Well-known redirect | `server/internal/caldav/wellknown.go` | `/.well-known/caldav` |
| Integration tests | `server/internal/caldav/handler_test.go` | CalDAV endpoint tests |

**CalDAV Requirements:**
- Mounted at `/dav` (not `/calendars`)
- `/.well-known/caldav` redirects to CalDAV base
- HTTP Basic auth with env vars `CALENDARAPP_USER`/`CALENDARAPP_PASS`
- Dev defaults `testuser`/`testpass` if env vars unset (with warning log)
- ETag = quoted SHA-256 of persisted ICS bytes: `"abc123..."`
- ETag mismatch returns HTTP 412 Precondition Failed
- PROPPATCH returns safe response (no-op or minimal) for Apple Calendar

### Additional Files

- `server/go.sum` — Generated dependency lock file (should be present)
- `server/go.mod` — May have dependency updates

---

## 3) Review Focus & Checklist

Rate each item as **PASS**, **WARN**, or **FAIL**.

### A) Repository Hygiene / Security

| Check | Criteria |
|-------|----------|
| No secrets committed | No API keys, tokens, passwords beyond documented dev defaults |
| `.gitignore` complete | Blocks `.env*`, `node_modules/`, `*.db`, build outputs |
| No agent state | `.codex/`, `.claude/`, `.agentvibes/` not committed (may be in .gitignore) |
| Dev defaults safe | `testuser`/`testpass` only used when env vars unset, logged as warning |

### B) Schema / Migrations

| Check | Criteria |
|-------|----------|
| Migration format | Goose `-- +goose Up` / `-- +goose Down` directives present |
| Events table | `ics TEXT NOT NULL` column exists |
| ETag column | `etag TEXT NOT NULL` column exists |
| Calendar uniqueness | Unique index/constraint on `(user_id, name)` |
| Foreign keys | Proper CASCADE delete on `calendars.user_id`, `events.calendar_id` |

### C) CalDAV Correctness

| Check | Criteria |
|-------|----------|
| Mount point | CalDAV handler mounted at `/dav` |
| Well-known | `/.well-known/caldav` redirects/routes to CalDAV base |
| Auth env override | `CALENDARAPP_USER`/`CALENDARAPP_PASS` override works end-to-end |
| No hardcoded user | No hardcoded `testuser` paths that break custom usernames |
| ETag format | Quoted SHA-256 of persisted ICS: `"hexstring..."` |
| 412 conflicts | ETag mismatch returns 412 Precondition Failed |
| PROPPATCH safe | Returns 207 or similar without breaking Apple Calendar setup |
| Chi methods | PROPFIND, PROPPATCH, REPORT, etc. registered with Chi |

### D) Code Quality

| Check | Criteria |
|-------|----------|
| Transactions | Write operations use transactions for atomicity |
| Error handling | Errors wrapped with context, not swallowed |
| No data loss | No obvious paths where events could be silently lost |
| Test coverage | Integration tests cover happy path + error cases |
| RFC compliance | ICS handling respects RFC 5545 (DTSTAMP, PRODID required) |

### E) Build & Tests (if runnable)

| Check | Criteria |
|-------|----------|
| `go test ./...` | All tests pass |
| `go vet ./...` | No significant vet issues |
| `go build` | Binary compiles successfully |
| Frontend | `pnpm install && pnpm lint && pnpm build` passes |

---

## 4) How to Validate

### Static Review (Minimum)

If you cannot execute code, perform static analysis:

1. **Read the migration file** — Verify schema meets requirements
2. **Check auth.go** — Verify env var handling, no hardcoded paths
3. **Check backend.go** — Verify ETag generation uses SHA-256 of ICS bytes
4. **Check handler_test.go** — Verify test coverage for PROPFIND, PUT, GET, DELETE, 412, PROPPATCH
5. **Search for secrets** — `grep -r "api[_-]?key\|token\|secret\|password\|sk-" .`

### Runnable Validation (If Available)

```bash
# Clone and checkout PR
gh pr checkout 4

# Backend verification
cd server
go test ./... -v
go vet ./...
go build ./cmd/calendarapp

# Frontend smoke test
cd ../web
pnpm install
pnpm lint
pnpm build

# Optional: Docker build
cd ..
docker build -f docker/Dockerfile .

# Optional: Run server for manual CalDAV testing
cd server
CALENDARAPP_USER=testuser CALENDARAPP_PASS=testpass go run ./cmd/calendarapp
# Then test with curl or a CalDAV client at http://localhost:8080/dav/
```

### Key Files to Review

| File | Focus |
|------|-------|
| `server/migrations/00001_init.sql` | Schema correctness |
| `server/internal/caldav/auth.go` | Env var handling, no hardcoded users |
| `server/internal/caldav/backend.go` | ETag generation, path parsing |
| `server/internal/caldav/handler.go` | Chi method registration, route setup |
| `server/internal/caldav/handler_test.go` | Test coverage completeness |
| `server/internal/data/sqlite_event_repo.go` | ICS storage, ETag handling |
| `server/cmd/calendarapp/main.go` | User bootstrap, route mounting |

---

## 5) Output Required

Post a **PR comment on PR #4** with the following structure:

```markdown
## GPT Pro Review: PR #4 Phase A+B CalDAV Implementation

**Date:** [Current Date]
**Reviewer:** GPT Pro Team
**PRP:** `docs/prp/gpt-pro-review-pr-4-phase-ab-caldav.md`

---

### Verdict: [PASS | WARN | FAIL]

---

### Checklist Results

#### A) Repository Hygiene / Security
- [ ] No secrets committed: [PASS/WARN/FAIL]
- [ ] `.gitignore` complete: [PASS/WARN/FAIL]
- [ ] No agent state committed: [PASS/WARN/FAIL]
- [ ] Dev defaults safe: [PASS/WARN/FAIL]

#### B) Schema / Migrations
- [ ] Migration format (goose): [PASS/WARN/FAIL]
- [ ] `events.ics TEXT NOT NULL`: [PASS/WARN/FAIL]
- [ ] `events.etag TEXT NOT NULL`: [PASS/WARN/FAIL]
- [ ] Calendar uniqueness: [PASS/WARN/FAIL]
- [ ] Foreign keys: [PASS/WARN/FAIL]

#### C) CalDAV Correctness
- [ ] Mount at `/dav`: [PASS/WARN/FAIL]
- [ ] Well-known redirect: [PASS/WARN/FAIL]
- [ ] Auth env override: [PASS/WARN/FAIL]
- [ ] No hardcoded user: [PASS/WARN/FAIL]
- [ ] ETag format (quoted SHA-256): [PASS/WARN/FAIL]
- [ ] 412 on conflicts: [PASS/WARN/FAIL]
- [ ] PROPPATCH safe: [PASS/WARN/FAIL]
- [ ] Chi methods registered: [PASS/WARN/FAIL]

#### D) Code Quality
- [ ] Transactions for writes: [PASS/WARN/FAIL]
- [ ] Error handling: [PASS/WARN/FAIL]
- [ ] No data loss risks: [PASS/WARN/FAIL]
- [ ] Test coverage: [PASS/WARN/FAIL]
- [ ] RFC 5545 compliance: [PASS/WARN/FAIL]

#### E) Build & Tests (if run)
- [ ] `go test ./...`: [PASS/WARN/FAIL/SKIP]
- [ ] `go vet ./...`: [PASS/WARN/FAIL/SKIP]
- [ ] `go build`: [PASS/WARN/FAIL/SKIP]
- [ ] Frontend build: [PASS/WARN/FAIL/SKIP]

---

### Blocking Issues

[List any FAIL items with file paths and line numbers if applicable]

Example:
- **FAIL:** Hardcoded `testuser` in `server/cmd/calendarapp/main.go:85` breaks custom usernames

---

### Non-Blocking Suggestions (max 5)

1. [Suggestion 1]
2. [Suggestion 2]
...

---

### Merge Recommendation

[One of:]
- **APPROVE:** All checks pass, ready to merge
- **APPROVE WITH RESERVATIONS:** Minor issues noted but not blocking
- **REQUEST CHANGES:** Blocking issues must be resolved before merge
```

---

## 6) Non-Goals (Do NOT Block On)

- Phase C features (AI planning, constraint engine)
- Todoist integration
- Full CalDAV scheduling/iTIP (RFC 6638)
- RFC 6578 `sync-collection` REPORT (post-MVP)
- Frontend features beyond build verification
- Performance optimization

---

## 7) Reference Documents

- **Implementation PRP:** `docs/prp/claude-dev-prp-mvp-sequence-caldav-repo-planner.md`
- **Codex Verification PRP:** `docs/prp/codex-verify-phase-a-b-caldav-repo.md`
- **PRD:** `_bmad-output/planning-artifacts/prd.md`
- **Architecture:** `_bmad-output/planning-artifacts/architecture.md`

---

## 8) Quick Reference

### ETag Behavior

```
PUT /dav/calendars/user/cal/event.ics
  → Server stores ICS bytes
  → Server computes ETag = sha256(stored_ics_bytes)
  → Server returns ETag header: "abc123..."

GET /dav/calendars/user/cal/event.ics
  → Server returns stored ICS bytes + ETag header

PUT with If-Match: "abc123..."
  → If current ETag matches: Update succeeds, new ETag returned
  → If current ETag differs: 412 Precondition Failed
```

### Auth Flow

```
1. Client requests /dav/calendars/...
2. Server returns 401 with WWW-Authenticate: Basic
3. Client sends Authorization: Basic base64(user:pass)
4. Server checks against CALENDARAPP_USER/CALENDARAPP_PASS env vars
5. If match: Request proceeds
6. If no match: 401 Unauthorized
```

### go-webdav Prefix Behavior

```
caldav.Handler configured with Prefix: "/dav"
  → Receives request: /dav/calendars/user/cal/...
  → Strips prefix before calling backend
  → Backend receives: /calendars/user/cal/...

Path parsing must handle BOTH forms if used elsewhere.
```

---

**END OF PRP**
