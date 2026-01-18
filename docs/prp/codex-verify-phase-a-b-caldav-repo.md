# PRP: Verify Phase A+B — Domain/Repo + CalDAV CRUD

**Type:** Verification Prompt (Codex Team)  
**Repo:** `airplne/calendar-app`  
**PR Under Review:** <ADD_PR_URL_OR_NUMBER>  
**Date:** 2026-01-18  
**Related Implementation PRP:** `docs/prp/claude-dev-prp-mvp-sequence-caldav-repo-planner.md`

---

## 1) Context

Calendar-app is a self-hosted CalDAV server (source of truth) with a web UI. We are executing the MVP sequence where **CalDAV multi-client sync correctness** is proven before building AI planning.

This PR claims to complete:

- **Phase A:** Domain model + repository layer (SQLite) + tests
- **Phase B:** CalDAV CRUD + ETag conflict handling + well-known discovery + tests

The PR should align with locked MVP stack decisions:

- **Backend:** Go 1.22+, `chi`, `emersion/go-webdav`, SQLite (`github.com/glebarez/sqlite`), goose migrations, `slog`
- **Frontend:** React SPA (Vite), HashRouter, Tailwind, headless primitives (copy-paste)
- **Real-time:** SSE at `/events` (no WebSockets requirement)
- **Deployment:** single binary with embedded web assets + Docker (distroless)

---

## 2) What You’re Verifying

### A) Phase A deliverables exist and are correct

Expected (names may vary, but responsibilities must exist):

- Domain models (events/calendars/tasks) and repository interfaces
- SQLite data layer: DB open + migrations + transaction helpers
- SQLite repos:
  - events CRUD with optimistic concurrency via **ETag precondition**
  - calendars CRUD (incl. `sync_token` as internal revision token)
  - users (single-user MVP support)
- Tests for domain + data layers

### B) Phase B deliverables exist and are correct

- CalDAV handler is **not** a stub (no “501 not implemented” handler)
- CalDAV is backed by `emersion/go-webdav` (CalDAV/WebDAV foundations)
- **HTTP Basic auth** for CalDAV endpoints with:
  - Env vars `CALENDARAPP_USER` / `CALENDARAPP_PASS`
  - Dev defaults `testuser` / `testpass` if unset (warn in logs if defaults used)
- **ETag behavior**
  - ETag is `sha256(icsBytesPersisted)` and returned as a quoted HTTP ETag
  - Conflicts return **HTTP 412 Precondition Failed**
- Apple Calendar compatibility:
  - PROPPATCH handled (no-op or safe response that doesn’t break setup)
- Well-known discovery:
  - `/.well-known/caldav` routes to the CalDAV base

### C) Schema matches the Phase A/B storage contract

Specifically:

- `events` table must store **full ICS** (e.g., `ics TEXT NOT NULL`) for roundtrip fidelity.
- ETag must be derived from the **persisted** ICS bytes (whatever is stored in `events.ics`).
- Extracted metadata columns may exist (summary/start/end/rrule/etc.) but must not replace the raw ICS source-of-truth column.

---

## 3) Non‑Goals (Do NOT Block On)

- Phase C (deterministic constraint engine + planning API stub) unless included in this PR explicitly
- Todoist integration
- Full CalDAV scheduling/iTIP (RFC 6638)
- RFC 6578 `sync-collection` REPORT support (post-MVP; `sync_token` is internal revision token only)

---

## 4) Review Checklist (PASS/WARN/FAIL)

### A) Repo Hygiene / Safety

- [ ] No secrets committed (tokens, keys, passwords beyond documented dev defaults)
- [ ] `.gitignore` blocks `.env*`, `node_modules/`, build outputs, SQLite DB files
- [ ] No local agent state committed (`.codex/`, `.claude/`, `.agentvibes/`) beyond intentionally tracked files

### B) Schema / Migrations

- [ ] Goose migrations are present and apply cleanly
- [ ] `events` table includes `ics TEXT NOT NULL` (or equivalent)
- [ ] `events` table includes `etag` (NOT NULL) and is updated on every write
- [ ] Any migration updates are consistent (either edit initial migration if DB is disposable, or add a new migration if preserving DB)

### C) Go Build & Tests

- [ ] `cd server && go test ./...` passes
- [ ] `cd server && go vet ./...` passes (or no significant vet issues)
- [ ] `cd server && go build ./cmd/calendarapp` succeeds

### D) CalDAV Behavior (Automated)

- [ ] Integration tests exist for: PROPFIND/REPORT (as implemented), PUT, GET, DELETE
- [ ] ETag mismatch path returns 412
- [ ] (Optional) Roundtrip test verifies ICS returned equals persisted ICS bytes (or document if canonicalization occurs and why)
- [ ] PROPPATCH does not break account setup flows
- [ ] `/.well-known/caldav` works (redirect/route)

### E) Web (Smoke)

- [ ] `cd web && pnpm install`
- [ ] `cd web && pnpm build`
- [ ] `cd web && pnpm lint` (if configured)

### F) Docker (if available)

- [ ] `docker build -f docker/Dockerfile .` succeeds
- [ ] Runtime image is distroless (or equivalently minimal) and runs as non-root

---

## 5) Suggested Commands

Checkout PR:
- `gh pr checkout <PR_NUMBER>`
  - or: `git fetch origin pull/<PR_NUMBER>/head:pr-<PR_NUMBER> && git checkout pr-<PR_NUMBER>`

Quick secret scan:
- `rg -n "(api[_-]?key|token|secret|password|sk-[A-Za-z0-9]|xai-|anthropic|openai)" . || true`

Backend:
- `cd server && go test ./...`
- `cd server && go vet ./...`
- `cd server && go build ./cmd/calendarapp`

Frontend:
- `cd web && pnpm install`
- `cd web && pnpm build`
- `cd web && pnpm lint`

Optional: run server (for manual checks):
- `cd server && CALENDARAPP_USER=testuser CALENDARAPP_PASS=testpass go run ./cmd/calendarapp`

---

## 6) Manual Interop Checklist (Optional but High Value)

If you can run the server, test at least 2 clients (3 is ideal):

- Apple Calendar (macOS/iOS)
- Fantastical
- DAVx⁵ (Android)

Suggested CalDAV URL pattern:
- `http://localhost:8080/dav/` (or the specific calendars URL the server advertises)

Verify:
- Create/edit/delete event on each client and confirm propagation
- Conflict test: edit same event from two clients; verify one side receives conflict behavior (server 412 / client re-fetch)

---

## 7) Output Required From Codex Team

Post a PR comment with:

1) **Verdict:** PASS / WARN / FAIL  
2) **Checklist results:** bullets of what passed/failed  
3) **Blocking issues** (if any) with exact file paths  
4) **Non-blocking suggestions** (optional, ≤5 bullets)  

If you cannot run Go/Docker in your environment, still complete a static review and mark missing tooling as WARN (not FAIL) unless the repo itself is clearly broken.
