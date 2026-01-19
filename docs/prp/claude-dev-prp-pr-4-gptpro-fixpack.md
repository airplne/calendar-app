# PRP: Claude Dev Team — Address GPT Pro Review Findings on PR #4

**Type:** Remediation Prompt (Claude Dev Team)
**Repo:** `airplne/calendar-app`
**PR:** [#4](https://github.com/airplne/calendar-app/pull/4)
**GPT Pro Verdict:** FAIL
**Date:** 2026-01-19
**Status:** Triage Required

---

## Executive Summary

GPT Pro's FAIL verdict on PR #4 cited:
1. **Missing `events.ics TEXT NOT NULL`** — Blocking
2. **Missing calendar uniqueness `(user_id, name)`** — Blocking
3. **"Data loss risk"** — Quality concern
4. **RFC 5545 WARN** — Quality concern

**Critical Finding:** GPT Pro's citations reference commit `a458497` on `main`, NOT the PR head. Initial verification confirms **both "blocking" issues ARE present in PR #4** (`phase-ab-caldav` branch). The FAIL verdict appears to be based on reviewing the wrong branch.

This PRP guides the dev team through:
1. Confirming the triage (Step 0)
2. Responding to GPT Pro with evidence (if triage confirms issues exist in PR)
3. Quality hardening for RFC 5545 compliance (regardless of triage outcome)

---

## 1) Context & Goal

### Background

- CalDAV is the calendar source of truth; Phase A+B proves multi-client sync correctness before AI planning
- PR #4 implements Phase A+B: domain models, SQLite repos, CalDAV CRUD with ETag conflict handling
- GPT Pro was asked to review PR #4 but their citations point to `main` branch commit `a458497`

### Goal

1. **Resolve the FAIL verdict** — Either by proving GPT Pro reviewed the wrong branch OR by fixing actual missing code
2. **Achieve GPT Pro PASS** — On correct review of PR #4
3. **Harden quality** — Address RFC 5545 WARN and data loss concerns with additional tests

### Scope

**In Scope:**
- Items from GPT Pro report (blocking issues, RFC 5545 WARN, data loss risk)
- ICS roundtrip fidelity tests

**Out of Scope:**
- Phase C AI planning
- Todoist integration
- Multi-user features
- Any changes not directly related to GPT Pro findings

---

## 2) Step 0: Triage — Verify PR #4 Branch State

**CRITICAL: Execute this step first before any code changes.**

### Triage Task 1: Verify `events.ics TEXT NOT NULL`

**File:** `server/migrations/00001_init.sql`
**Expected Location:** Lines 30-52 (events table definition)
**Expected Content:**

```sql
-- Events table (iCalendar VEVENT components)
-- Hybrid storage: full ICS for CalDAV fidelity + extracted metadata for queries
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    calendar_id INTEGER NOT NULL,
    uid TEXT NOT NULL,  -- iCalendar UID (globally unique)
    ics TEXT NOT NULL,  -- Full VEVENT component (stored as-is for CalDAV roundtrip)
    ...
```

**Verification Command:**
```bash
git checkout phase-ab-caldav
grep -n "ics TEXT NOT NULL" server/migrations/00001_init.sql
```

**Expected Output:**
```
36:    ics TEXT NOT NULL,  -- Full VEVENT component (stored as-is for CalDAV roundtrip)
```

**Triage Result:**
- [ ] **PRESENT** — Column exists in PR #4 → GPT Pro reviewed wrong branch
- [ ] **ABSENT** — Column missing → Implement fix per Section 3A

---

### Triage Task 2: Verify Calendar Uniqueness Constraint

**File:** `server/migrations/00001_init.sql`
**Expected Location:** Lines 27-28
**Expected Content:**

```sql
-- Enforce uniqueness: one calendar per (user_id, name) combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendars_user_id_name ON calendars(user_id, name);
```

**Verification Command:**
```bash
git checkout phase-ab-caldav
grep -n "idx_calendars_user_id_name" server/migrations/00001_init.sql
```

**Expected Output:**
```
28:CREATE UNIQUE INDEX IF NOT EXISTS idx_calendars_user_id_name ON calendars(user_id, name);
```

**Triage Result:**
- [ ] **PRESENT** — Index exists in PR #4 → GPT Pro reviewed wrong branch
- [ ] **ABSENT** — Index missing → Implement fix per Section 3A

---

### Triage Task 3: Compare main vs PR branch

**Verification Command:**
```bash
# Show what main has (should NOT have ics column or uniqueness index)
git show main:server/migrations/00001_init.sql | grep -E "(ics TEXT|idx_calendars_user_id_name)"

# Show what PR branch has (SHOULD have both)
git show phase-ab-caldav:server/migrations/00001_init.sql | grep -E "(ics TEXT|idx_calendars_user_id_name)"
```

**Expected Result:**
- `main`: No output (features missing)
- `phase-ab-caldav`: Both lines present

---

### Triage Decision Matrix

| Triage Result | Action |
|---------------|--------|
| Both PRESENT in PR #4 | Proceed to **Section 3B** (Respond to GPT Pro) |
| One or both ABSENT | Proceed to **Section 3A** (Implement Fixes) |

---

## 3A) If Issues ARE Missing (True FAIL) — Implement Fixes

**Skip this section if triage confirms both items exist in PR #4.**

### Fix 1: Add `events.ics TEXT NOT NULL`

**File:** `server/migrations/00001_init.sql`

Add the `ics` column to the events table:

```sql
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    calendar_id INTEGER NOT NULL,
    uid TEXT NOT NULL,
    ics TEXT NOT NULL,  -- Full VEVENT component for CalDAV roundtrip
    -- ... rest of columns
```

### Fix 2: Add Calendar Uniqueness Constraint

**File:** `server/migrations/00001_init.sql`

Add after the calendars table definition:

```sql
-- Enforce uniqueness: one calendar per (user_id, name) combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendars_user_id_name ON calendars(user_id, name);
```

### Fix 3: Update Repository to Use ICS Column

**File:** `server/internal/data/sqlite_event_repo.go`

Ensure all CRUD operations:
1. Store full ICS in the `ics` column
2. Derive ETag from `sha256(ics_bytes)`
3. Return stored ICS bytes unchanged on GET

### Fix 4: Add Repository Tests

**File:** `server/internal/data/sqlite_event_repo_test.go`

Add test verifying:
1. ICS is stored and retrieved unchanged
2. ETag matches SHA-256 of stored ICS
3. Calendar uniqueness constraint triggers `domain.ErrConflict`

---

## 3B) If Issues ARE Present (GPT Pro Reviewed Wrong Branch) — Respond & Request Re-Review

**This is the expected scenario based on initial verification.**

### Step 1: Post Evidence Comment on PR #4

Use this template to respond to GPT Pro's review:

```markdown
## Response to GPT Pro Review — Request for Re-Review

Thank you for the detailed review. After investigating the FAIL findings, we've identified that **the review appears to have been conducted against the `main` branch rather than PR #4's `phase-ab-caldav` branch**.

> **Please review PR #4 (branch `phase-ab-caldav`)** — the cited commit `a458497` is on `main` and only contains the PRP, not the Phase A+B implementation.
>
> **Note:** PR head may advance with docs-only commits. To review the Phase A+B implementation, use `gh pr checkout 4` or view the "Files changed" tab on GitHub. The core CalDAV CRUD implementation is introduced in commit `927a19a`.

### Evidence

**GPT Pro cited commit:** `a458497` (on `main`)
**PR #4 branch:** `phase-ab-caldav`
**Phase A+B implementation commit:** `927a19a` (on `phase-ab-caldav`)

The cited "missing" items **ARE present in PR #4**:

#### 1. `events.ics TEXT NOT NULL` — PRESENT

**File:** `server/migrations/00001_init.sql:36`
```sql
ics TEXT NOT NULL,  -- Full VEVENT component (stored as-is for CalDAV roundtrip)
```

[View in PR #4 Files Changed](https://github.com/airplne/calendar-app/pull/4/files)

#### 2. Calendar Uniqueness Constraint — PRESENT

**File:** `server/migrations/00001_init.sql:27-28`
```sql
-- Enforce uniqueness: one calendar per (user_id, name) combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendars_user_id_name ON calendars(user_id, name);
```

[View in PR #4 Files Changed](https://github.com/airplne/calendar-app/pull/4/files)

### Verification Commands

To review the correct branch:
```bash
gh pr checkout 4
# OR
git fetch origin pull/4/head:pr-4 && git checkout pr-4

# Then verify:
grep -n "ics TEXT NOT NULL" server/migrations/00001_init.sql
grep -n "idx_calendars_user_id_name" server/migrations/00001_init.sql

# Run tests:
cd server && go test ./... -v
```

### Request

Could you please re-run the review by:
1. Checking out PR #4 (`gh pr checkout 4`) or reviewing the "Files changed" tab
2. Running `cd server && go test ./... && go vet ./...` if possible
3. Updating the verdict based on the actual PR content

The RFC 5545 WARN and "data loss risk" concerns are valid quality suggestions. If you still want extra hardening after re-review, we can add roundtrip fidelity tests for ATTENDEE/VALARM/X-* in a follow-up PR.

Thank you!
```

### Step 2: Update PR Description (Optional)

Add a "Schema Guarantees" section to PR description to prevent future misreads:

```markdown
## Schema Guarantees (Phase A+B)

This PR implements the following schema guarantees per the architecture doc:

| Requirement | Implementation | Location |
|-------------|----------------|----------|
| Full ICS storage | `events.ics TEXT NOT NULL` | `migrations/00001_init.sql:36` |
| ETag derivation | SHA-256 of persisted ICS bytes | `server/internal/domain/event.go:GenerateETag` |
| Calendar uniqueness | `UNIQUE INDEX (user_id, name)` | `migrations/00001_init.sql:28` |
| Conflict detection | 412 Precondition Failed on ETag mismatch | `caldav/backend.go:Put` |
```

---

## 4) Quality Hardening — RFC 5545 & Data Integrity

**Execute regardless of triage outcome to address GPT Pro's quality concerns.**

### 4.1: Add ICS Roundtrip Fidelity Test

**File:** `server/internal/data/sqlite_event_repo_test.go`

**Purpose:** Ensure unknown/arbitrary ICS properties survive PUT → GET roundtrip unchanged.

```go
func TestSQLiteEventRepo_ICS_Roundtrip_Fidelity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	// ICS with extra properties that must survive roundtrip
	icsWithExtras := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Roundtrip//EN
BEGIN:VEVENT
UID:roundtrip-test-1
DTSTAMP:20260119T120000Z
DTSTART:20260120T100000Z
DTEND:20260120T110000Z
SUMMARY:Roundtrip Test Event
DESCRIPTION:This event has extra properties
LOCATION:Test Location
ATTENDEE;CN=John Doe:mailto:john@example.com
ATTENDEE;CN=Jane Doe:mailto:jane@example.com
CATEGORIES:MEETING,IMPORTANT
X-CUSTOM-PROP:Custom Value
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Reminder
TRIGGER:-PT15M
END:VALARM
END:VEVENT
END:VCALENDAR`

	event := &domain.Event{
		CalendarID: 1,
		UID:        "roundtrip-test-1",
		ICS:        icsWithExtras,
		Summary:    "Roundtrip Test Event",
		StartTime:  time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 1, 20, 11, 0, 0, 0, time.UTC),
	}

	// Store event
	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Retrieve event
	retrieved, err := repo.GetByUID(ctx, 1, "roundtrip-test-1")
	if err != nil {
		t.Fatalf("GetByUID failed: %v", err)
	}

	// Verify ICS unchanged (byte-for-byte)
	if retrieved.ICS != icsWithExtras {
		t.Errorf("ICS roundtrip fidelity failed\nExpected:\n%s\n\nGot:\n%s", icsWithExtras, retrieved.ICS)
	}

	// Verify specific properties survived
	mustContain := []string{
		"ATTENDEE;CN=John Doe:mailto:john@example.com",
		"ATTENDEE;CN=Jane Doe:mailto:jane@example.com",
		"CATEGORIES:MEETING,IMPORTANT",
		"X-CUSTOM-PROP:Custom Value",
		"BEGIN:VALARM",
		"TRIGGER:-PT15M",
	}

	for _, prop := range mustContain {
		if !strings.Contains(retrieved.ICS, prop) {
			t.Errorf("Missing property in roundtrip: %s", prop)
		}
	}
}
```

### 4.2: Verify ETag Derivation from Persisted ICS

**File:** `server/internal/data/sqlite_event_repo_test.go`

Add assertion that ETag is derived from stored ICS bytes:

```go
func TestSQLiteEventRepo_ETag_Derived_From_ICS(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	icsData := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//ETag//EN
BEGIN:VEVENT
UID:etag-test-1
DTSTAMP:20260119T120000Z
DTSTART:20260120T100000Z
DTEND:20260120T110000Z
SUMMARY:ETag Test
END:VEVENT
END:VCALENDAR`

	event := &domain.Event{
		CalendarID: 1,
		UID:        "etag-test-1",
		ICS:        icsData,
		Summary:    "ETag Test",
		StartTime:  time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 1, 20, 11, 0, 0, 0, time.UTC),
	}

	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	retrieved, _ := repo.GetByUID(ctx, 1, "etag-test-1")

	// Compute expected ETag from stored ICS
	expectedETag := domain.GenerateETag([]byte(retrieved.ICS))

	if retrieved.ETag != expectedETag {
		t.Errorf("ETag not derived from ICS\nExpected: %s\nGot: %s", expectedETag, retrieved.ETag)
	}
}
```

### 4.3: Ensure Test ICS Includes Required RFC 5545 Properties

**File:** `server/internal/caldav/handler_test.go`

Audit all test ICS data to ensure:
- `VCALENDAR` has `PRODID`
- `VEVENT` has `DTSTAMP` and `UID`

**Checklist:**
- [ ] `TestCalDAV_PUT_CreateEvent` — Has PRODID, DTSTAMP, UID
- [ ] `TestCalDAV_GET_RetrieveEvent` — Has PRODID, DTSTAMP, UID
- [ ] `TestCalDAV_PUT_ETagConflict` — Has PRODID, DTSTAMP, UID
- [ ] `TestCalDAV_DELETE_Event` — Has PRODID, DTSTAMP, UID

### 4.4: Document ICS Handling Policy

Add comment to `server/internal/data/sqlite_event_repo.go`:

```go
// ICS Handling Policy:
//
// This repository implements STORE-AS-IS semantics for ICS data:
// 1. The full ICS blob is stored in events.ics without modification
// 2. ETag is computed as sha256(persisted_ics_bytes)
// 3. GET returns the exact bytes that were stored by PUT
// 4. Unknown properties, VALARM, ATTENDEE, X-* etc. are preserved
//
// We do NOT canonicalize or reformat ICS data. This ensures:
// - CalDAV client round-trip fidelity
// - Preservation of client-specific extensions
// - Compliance with RFC 5545 "MUST preserve" requirements
```

---

## 5) Validation & Acceptance Criteria

### Must Pass

```bash
# Backend tests
cd server && go test ./... -v

# Expected: All tests pass including new roundtrip fidelity test

# Static analysis
cd server && go vet ./...

# Expected: No issues

# Build
cd server && go build ./cmd/calendarapp

# Expected: Binary compiles successfully

# Frontend (smoke test)
cd web && pnpm install && pnpm lint && pnpm build

# Expected: All pass
```

### GPT Pro Re-Review Criteria

The PR is complete when GPT Pro:
1. **Acknowledges** the original FAIL was based on reviewing `main` instead of PR #4
2. **Updates verdict to PASS** after reviewing the correct branch
3. **OR** If GPT Pro finds NEW issues in the actual PR content, those are addressed

### No Regressions

- [ ] No new secrets committed
- [ ] No agent state directories committed (`.codex/`, `.claude/`, `.agentvibes/`)
- [ ] All existing tests still pass
- [ ] CalDAV 7-test suite passes:
  - `TestCalDAV_Unauthorized_Without_Auth`
  - `TestCalDAV_PROPFIND_ListCalendars`
  - `TestCalDAV_PUT_CreateEvent`
  - `TestCalDAV_GET_RetrieveEvent`
  - `TestCalDAV_PUT_ETagConflict`
  - `TestCalDAV_DELETE_Event`
  - `TestCalDAV_PROPPATCH_AppleCompatibility`

---

## 6) Output — What to Deliver

### If Triage Confirms Issues Exist in PR #4 (Expected Path)

1. **Post PR comment** using template from Section 3B Step 1
2. **Add roundtrip fidelity test** per Section 4.1
3. **Add ETag derivation test** per Section 4.2
4. **Verify all test ICS** has required RFC 5545 properties per Section 4.3
5. **Add ICS handling policy comment** per Section 4.4
6. **Request GPT Pro re-review** on correct branch

### If Triage Shows Issues ARE Missing (Unexpected Path)

1. **Implement fixes** per Section 3A
2. **Add tests** per Section 4
3. **Push commits** to PR #4
4. **Request GPT Pro re-review**

### Final PR Comment Template (After Quality Hardening)

```markdown
## Quality Hardening Complete — Ready for Re-Review

Per GPT Pro's quality suggestions, we've added:

### 1. ICS Roundtrip Fidelity Test
**File:** `server/internal/data/sqlite_event_repo_test.go`
- Tests that ATTENDEE, VALARM, X-* properties survive PUT → GET unchanged
- Verifies byte-for-byte ICS preservation

### 2. ETag Derivation Verification
- Confirms ETag = SHA-256(persisted ICS bytes)
- Added explicit test assertion

### 3. RFC 5545 Compliance Audit
- All test ICS now includes required properties (PRODID, DTSTAMP, UID)
- Added documentation of STORE-AS-IS policy

### Verification
```bash
cd server && go test ./... -v
# All tests pass including new fidelity tests
```

Please re-review PR #4 by checking out the branch:
```bash
gh pr checkout 4
```

Thank you for the thorough review that prompted these quality improvements!
```

---

## 7) Reference

### Key Files

| File | Purpose |
|------|---------|
| `server/migrations/00001_init.sql` | Schema definition — verify ics + uniqueness |
| `server/internal/data/sqlite_event_repo.go` | Event CRUD — verify ICS storage |
| `server/internal/data/sqlite_event_repo_test.go` | Add roundtrip + ETag tests |
| `server/internal/caldav/handler_test.go` | Verify RFC 5545 compliance in test data |

### Commits to Compare

| Branch | Commit | Content |
|--------|--------|---------|
| `main` | `a458497` | PRP only, no Phase A+B code |
| `phase-ab-caldav` (head) | `a5ccc6a` | Latest docs-only commit |
| `phase-ab-caldav` (impl) | `927a19a` | Full Phase A+B implementation |

### GPT Pro Original Findings

| Finding | Severity | Status in PR #4 |
|---------|----------|-----------------|
| Missing `events.ics TEXT NOT NULL` | Blocking | **PRESENT** (line 36) |
| Missing calendar uniqueness | Blocking | **PRESENT** (line 28) |
| "Data loss risk" | Quality | Address with roundtrip test |
| RFC 5545 WARN | Quality | Address with property audit |

---

**END OF PRP**
