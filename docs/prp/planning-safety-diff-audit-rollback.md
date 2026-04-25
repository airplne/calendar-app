# PRP: Planning Safety Skeleton, Diff, Audit Log, and Rollback

**Type:** Implementation PRP / Product Requirements Prompt  
**Target Team:** Backend + API + QA  
**Status:** Proposed  
**Priority:** MVP safety backbone

---

## 1. Context

Calendar-app's trust loop is:

```text
Generate proposal -> validate -> show diff -> confirm -> revalidate -> apply -> audit -> rollback
```

Before connecting real LLMs, Calendar-app needs a deterministic proposal, diff, validation, apply, audit, and rollback path. This PRP expands the existing Phase C planning skeleton into an implementation-ready safety backbone.

Current repo assumptions:

- CalDAV remains source of truth.
- Event writes must go through repository/CalDAV-compatible persistence.
- ETags are already part of event storage and conflict handling.
- The current `audit_log` table exists but may need extension for proposal-specific payloads.
- No real LLM or Todoist write is required here.

---

## 2. Problem statement

Calendar-app cannot safely let AI or planning logic edit a calendar until every proposed change is typed, validated, previewable, explicitly applied, auditable, and reversible. Without this skeleton, later LLM-generated changes would lack reliable safety rails and the UI could not honestly promise no silent writes.

---

## 3. Goals

- Define stable domain models for proposals, diffs, validation, audit, and rollback.
- Add a deterministic/mock daily planning endpoint that returns one validated proposal.
- Store proposals with expiration, affected event ETag references, and validation results.
- Ensure invalid proposals cannot be applied.
- Revalidate constraints and ETags immediately before apply.
- Apply event changes through the repository layer.
- Write audit log entries with rollback payloads.
- Implement conflict-aware rollback.
- Avoid partial apply whenever possible with transactions.

---

## 4. Non-goals

- No real LLM integration.
- No Todoist write integration.
- No auto-apply.
- No chat planning.
- No advanced recurrence edits.
- No message sending.
- No iTIP scheduling.
- No team scheduling.
- No broad preference learning beyond minimal focus protection metadata.

---

## 5. User stories

1. As a user, I can request a daily plan and receive a validated proposal with a clear diff.
2. As a user, I cannot apply a proposal that overlaps an immovable event or violates focus protection.
3. As a user, if my calendar changes after preview, apply is blocked and I am told to regenerate.
4. As a user, every applied proposal appears in the audit log with what changed and why.
5. As a user, I can rollback an applied proposal if affected events have not changed externally.
6. As a user, rollback is blocked with a clear explanation if an external client modified an affected event.

---

## 6. Functional requirements

### FR1 - Planning domain model

Add domain models under `server/internal/planner` or `server/internal/domain`.

Required types:

```go
type PlanProposal struct {
    ID               string
    UserID           int64
    Date             time.Time
    Status           ProposalStatus
    Source           ProposalSource
    Summary          string
    Explanation      string
    Changes          []ProposedChange
    ValidationResult ValidationResult
    AffectedRefs     []AffectedEntityRef
    ExpiresAt        time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type ProposedChange struct {
    ID                    string
    Type                  ChangeType
    EntityType            EntityType
    EntityID              string
    CalendarID            int64
    UID                   string
    Before                *ChangeSnapshot
    After                 *ChangeSnapshot
    HumanSummary          string
    Reason                string
    IsDestructive         bool
    RequiresStrongConfirm bool
}

type ValidationResult struct {
    Valid      bool
    CheckedAt  time.Time
    Violations []Violation
    Warnings   []ValidationWarning
}

type Violation struct {
    Type               ViolationType
    Severity           string
    Message            string
    ProposedChangeID   string
    ConflictingEntityID string
    ConflictingUID     string
}

type RollbackPayload struct {
    ProposalID          string
    AuditLogEntryID     int64
    ChangesToUndo       []RollbackChange
    AffectedRefsAtApply []AffectedEntityRef
    CreatedAt           time.Time
}

type AuditLogEntry struct {
    ID                   int64
    UserID               int64
    Action               string
    ProposalID           string
    Summary              string
    Changes              []ProposedChange
    RollbackPayload      *RollbackPayload
    Result               string
    ErrorCode            string
    RedactedErrorMessage string
    CreatedAt            time.Time
}
```

### FR2 - Diff format

Every proposal must expose machine-readable and human-readable diffs.

Supported change types:

- `add`
- `move`
- `delete`
- `update`
- `todoist_update` placeholder
- `message_draft` placeholder

For MVP, only `add` focus block must be fully applied. Other change types may validate/render but return `not_implemented` if applied. The deterministic mock should initially use only `add`.

Example:

```json
{
  "changes": [
    {
      "id": "chg_01",
      "type": "add",
      "entity_type": "event",
      "human_summary": "+ Add focus block: Deep Work, 09:30-11:30",
      "after": {
        "uid": "focus-20260425-0930",
        "summary": "Deep Work",
        "start_time": "2026-04-25T09:30:00-04:00",
        "end_time": "2026-04-25T11:30:00-04:00",
        "calendar_id": 1,
        "event_kind": "focus_block",
        "protection_level": "protected"
      },
      "reason": "Longest open morning slot inside working hours"
    }
  ]
}
```

### FR3 - Constraint engine

Implement constraints:

1. `TimeConflictConstraint`
   - No proposed add/move may overlap fixed confirmed events.
2. `TimeWindowConstraint`
   - Proposed focus block must fit configured working hours.
   - Default window: 09:00-17:00 local time.
3. `ImmovableEventConstraint`
   - Proposed move/delete/update cannot affect events tagged immovable.
   - Proposed add cannot overlap an immovable event.
4. `FocusBlockProtectionConstraint`
   - Proposed changes must not weaken or overlap protected focus blocks. For this PRP, block overlaps with protected focus blocks.
5. `MinimumFocusDurationConstraint`
   - Focus block must meet default minimum duration, initially 60 minutes unless preference exists.

Validation must happen:

- Before proposal response is returned.
- Immediately before apply.
- Immediately before rollback where relevant.

Invalid proposal behavior:

- API may return invalid proposal with violations for debugging, but `can_apply=false`.
- UI must not show actionable Apply for invalid proposal.
- Apply endpoint must reject invalid proposals regardless of UI behavior.

### FR4 - Proposal storage

Persist proposals long enough to apply or expire.

Requirements:

- Store proposal as JSON and normalized metadata.
- Default expiration: 15 minutes.
- Store status: `draft`, `validated`, `invalid`, `expired`, `applied`, `rejected`, `apply_failed`, `rolled_back`, `rollback_blocked`.
- Store affected event refs: `calendar_id`, `uid`, `etag_at_generation`, `start_time`, `end_time`.
- Expired proposals cannot apply.
- If affected ETags differ at apply time, proposal is stale.

### FR5 - Deterministic daily proposal

Implement:

```http
POST /api/v1/plan/daily
```

Behavior:

- Reads today's existing events.
- Finds the first or best safe focus block slot using deterministic rules.
- Default target: 2-hour block preferred, 60-minute minimum, within working hours, no overlaps, prefer morning slot.
- Returns `proposal_id`, `summary`, `explanation`, `changes`, `validation_result`, `can_apply`, and `expires_at`.
- No LLM required.

### FR6 - Proposal retrieval

Implement:

```http
GET /api/v1/plan/proposals/{proposal_id}
```

Returns stored proposal, current status, validation result, expiration, and can-apply state.

### FR7 - Apply endpoint

Implement:

```http
POST /api/v1/plan/apply
```

Request:

```json
{
  "proposal_id": "prop_123",
  "confirm_token": "optional-for-future"
}
```

Apply behavior:

1. Load proposal.
2. Reject if not found.
3. Reject if expired.
4. Reject if already applied/rolled back/rejected.
5. Re-read affected events.
6. Compare current ETags to stored `etag_at_generation`.
7. Re-run constraints on current calendar state.
8. If invalid, return violations and set proposal status `apply_failed` or `stale`.
9. Begin transaction.
10. Apply supported changes through repository layer.
11. Store rollback payload.
12. Write audit log entry.
13. Mark proposal `applied`.
14. Commit transaction.
15. Return applied summary.

Avoid partial apply:

- All supported event changes must be applied in one transaction.
- If any change fails, rollback the transaction.
- Future external integrations should use a separate saga/compensation design; out of scope here.

### FR8 - Rollback endpoint

Implement:

```http
POST /api/v1/plan/rollback
```

Request options:

```json
{
  "proposal_id": "prop_123"
}
```

or:

```json
{
  "audit_log_entry_id": 42
}
```

Rollback behavior:

1. Load applied audit entry and rollback payload.
2. Re-read all affected current events.
3. Compare current ETags to `AffectedRefsAtApply`.
4. If any ETag changed externally, block rollback.
5. Explain which event changed and why rollback is blocked.
6. If safe, begin transaction.
7. Undo applied changes:
   - Added event -> delete event.
   - Moved event -> restore previous start/end/ICS.
   - Updated event -> restore previous snapshot.
   - Deleted event -> recreate previous event.
8. Write rollback audit log entry.
9. Mark original proposal `rolled_back`.
10. Commit transaction.
11. Return rollback summary.

### FR9 - Audit log

Extend existing `audit_log` model if needed.

Audit entries must include:

- User ID.
- Action.
- Proposal ID.
- Human summary.
- Machine-readable changes.
- Rollback payload.
- Result.
- Created timestamp.
- Redacted errors.

Expose:

```http
GET /api/v1/audit-log
GET /api/v1/audit-log/{entry_id}
```

### FR10 - Rejection endpoint

Needed by the First Aha UX PRP, but can be backend-ready here:

```http
POST /api/v1/plan/proposals/{proposal_id}/reject
```

Request:

```json
{
  "reason_code": "bad_time",
  "free_text": "Too close to lunch"
}
```

Supported reason codes:

- `bad_time`
- `too_long`
- `too_short`
- `conflicts_with_preference`
- `not_enough_context`
- `other`

---

## 7. Non-functional requirements

- Determinism: same calendar state + same preferences = same mock proposal.
- Safety: apply always revalidates.
- Atomicity: calendar mutations and audit write happen in the same DB transaction where possible.
- Privacy: audit entries and diffs should avoid raw ICS in default API responses; raw snapshots may be stored for rollback but not exposed by default.
- Latency: deterministic daily proposal should return under 500ms for typical single-user calendars.
- Reliability: stale proposals must never apply.
- Testability: constraint engine must be unit-testable without HTTP server.
- Extensibility: LLM proposal source can later create the same `PlanProposal` format.

---

## 8. Technical approach

### Backend architecture

Suggested packages:

```text
server/internal/planner
  types.go
  constraints.go
  time_conflict.go
  time_window.go
  immovable.go
  focus_protection.go
  slot_finder.go
  proposal_service.go
  apply_service.go
  rollback_service.go

server/internal/api
  plan.go
  audit_log.go

server/internal/data
  sqlite_plan_proposal_repo.go
  sqlite_audit_log_repo.go
```

Service interfaces:

```go
type ProposalService interface {
    GenerateDaily(ctx context.Context, userID int64, date time.Time) (*PlanProposal, error)
    Get(ctx context.Context, userID int64, proposalID string) (*PlanProposal, error)
    Reject(ctx context.Context, userID int64, proposalID string, reason RejectionReason) error
}

type ApplyService interface {
    Apply(ctx context.Context, userID int64, proposalID string) (*ApplyResult, error)
}

type RollbackService interface {
    Rollback(ctx context.Context, userID int64, req RollbackRequest) (*RollbackResult, error)
}

type ConstraintEngine interface {
    Validate(ctx context.Context, proposal *PlanProposal, state CalendarState) (*ValidationResult, error)
}
```

### Validation flow

```text
Generate candidate
-> Build CalendarState from EventRepo
-> Validate constraints
-> Store proposal with validation result and affected refs
-> Return proposal with can_apply = validation.valid && not expired
```

### Apply flow

```text
Load proposal
-> Check status/expiration
-> Re-read affected events
-> Check generation ETags
-> Rebuild CalendarState
-> Revalidate constraints
-> Begin transaction
-> Apply event changes
-> Store rollback payload
-> Write audit log
-> Mark proposal applied
-> Commit
```

### Rollback flow

```text
Load audit entry
-> Load rollback payload
-> Re-read affected events
-> Compare ETags at apply time
-> If safe, begin transaction
-> Undo event changes
-> Write rollback audit entry
-> Mark proposal rolled_back
-> Commit
```

---

## 9. Files likely to change

Backend:

```text
server/internal/planner/types.go
server/internal/planner/constraints.go
server/internal/planner/time_conflict.go
server/internal/planner/time_window.go
server/internal/planner/immovable.go
server/internal/planner/focus_protection.go
server/internal/planner/slot_finder.go
server/internal/planner/proposal_service.go
server/internal/planner/apply_service.go
server/internal/planner/rollback_service.go
server/internal/planner/*_test.go
server/internal/api/plan.go
server/internal/api/audit_log.go
server/internal/api/types.go
server/internal/data/sqlite_plan_proposal_repo.go
server/internal/data/sqlite_audit_log_repo.go
server/internal/domain/event.go
server/internal/domain/repository.go
server/migrations/00003_planning_safety.sql
server/cmd/calendarapp/main.go
```

Frontend-ready types only:

```text
web/src/lib/types.ts
web/src/lib/api.ts
```

Docs:

```text
docs/architecture/planning-safety.md
docs/api/planning.md
```

---

## 10. API changes

```http
POST /api/v1/plan/daily
GET  /api/v1/plan/proposals/{proposal_id}
POST /api/v1/plan/proposals/{proposal_id}/reject
POST /api/v1/plan/apply
POST /api/v1/plan/rollback
GET  /api/v1/audit-log
GET  /api/v1/audit-log/{entry_id}
```

Common error response:

```json
{
  "error": "stale_proposal",
  "message": "This proposal is stale because the calendar changed after it was generated.",
  "details": {
    "proposal_id": "prop_123",
    "changed_refs": [
      {
        "uid": "event_hash_or_uid",
        "reason": "etag_changed"
      }
    ]
  }
}
```

---

## 11. Database/schema changes

Create migration `server/migrations/00003_planning_safety.sql`.

```sql
CREATE TABLE plan_proposals (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL,
  date DATE NOT NULL,
  status TEXT NOT NULL,
  source TEXT NOT NULL,
  summary TEXT,
  explanation TEXT,
  changes_json TEXT NOT NULL,
  validation_json TEXT NOT NULL,
  affected_refs_json TEXT NOT NULL,
  rejection_json TEXT,
  expires_at DATETIME NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_plan_proposals_user_date ON plan_proposals(user_id, date);
CREATE INDEX idx_plan_proposals_status ON plan_proposals(status);
CREATE INDEX idx_plan_proposals_expires_at ON plan_proposals(expires_at);

CREATE TABLE rollback_payloads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  proposal_id TEXT NOT NULL,
  audit_log_entry_id INTEGER,
  payload_json TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (proposal_id) REFERENCES plan_proposals(id) ON DELETE CASCADE,
  FOREIGN KEY (audit_log_entry_id) REFERENCES audit_log(id) ON DELETE SET NULL
);

CREATE INDEX idx_rollback_payloads_proposal_id ON rollback_payloads(proposal_id);

ALTER TABLE audit_log ADD COLUMN proposal_id TEXT;
ALTER TABLE audit_log ADD COLUMN result TEXT;
ALTER TABLE audit_log ADD COLUMN error_code TEXT;
ALTER TABLE audit_log ADD COLUMN redacted_error_message TEXT;
```

---

## 12. UX changes, if applicable

Backend-only PRP, but API must support future UI:

- `can_apply` boolean.
- `validation_result.valid`.
- `validation_result.violations`.
- Grouped diffs by change type.
- Rollback availability.
- Rollback conflict explanation.
- Audit log links.

---

## 13. Test plan

Constraint tests:

- `TestTimeConflictConstraint_NoOverlap`.
- `TestTimeConflictConstraint_Overlap`.
- `TestTimeWindowConstraint_InsideWindow`.
- `TestTimeWindowConstraint_OutsideWindow`.
- `TestImmovableConstraint_BlocksOverlap`.
- `TestImmovableConstraint_BlocksMove`.
- `TestFocusProtectionConstraint_BlocksOverlap`.
- `TestMinimumFocusDurationConstraint_BlocksTooShort`.

Proposal tests:

- `TestPlanDaily_ReturnsDeterministicProposal`.
- `TestPlanDaily_ReturnsInvalidWhenNoSafeSlot`.
- `TestProposalExpires`.
- `TestStoredProposalIncludesAffectedETags`.

Apply tests:

- `TestApply_RevalidatesBeforeWrite`.
- `TestApply_RejectsInvalidProposal`.
- `TestApply_RejectsExpiredProposal`.
- `TestApply_RejectsStaleETag`.
- `TestApply_WritesEventViaRepository`.
- `TestApply_WritesAuditLog`.
- `TestApply_StoresRollbackPayload`.
- `TestApply_TransactionRollbackOnFailure`.

Rollback tests:

- `TestRollback_DeletesAddedFocusBlock`.
- `TestRollback_RestoresMovedEvent`.
- `TestRollback_BlocksWhenExternalETagChanged`.
- `TestRollback_WritesAuditLogEntry`.
- `TestRollback_CannotRollbackTwice`.

API tests:

- `TestPostPlanDaily`.
- `TestGetProposal`.
- `TestPostApply`.
- `TestPostRollback`.
- `TestGetAuditLog`.
- `TestRejectProposal`.

---

## 14. Manual validation steps

1. Run `cd server && go test ./...`.
2. Start server.
3. Seed calendar with fixed events at 09:00-09:30 and 12:00-13:00.
4. Call `POST /api/v1/plan/daily` with today's date.
5. Confirm response proposes a non-overlapping focus block.
6. Apply proposal.
7. Confirm event exists in `events`.
8. Confirm audit entry exists.
9. Rollback proposal.
10. Confirm event removed/restored.
11. Generate another proposal.
12. Modify affected event externally.
13. Attempt apply.
14. Confirm stale proposal error.
15. Apply a proposal, externally modify focus event, then attempt rollback.
16. Confirm rollback conflict error.

---

## 15. Acceptance criteria

- `go test ./...` passes.
- Constraint tests cover overlap, allowed windows, immovable events, and protected focus blocks.
- `/api/v1/plan/daily` can return a deterministic/mock validated proposal.
- Proposal includes machine-readable changes and human-readable diff summaries.
- Invalid proposals return violations and cannot be applied.
- `/api/v1/plan/apply` revalidates before write.
- Apply checks affected event ETags against proposal generation refs.
- Apply writes event changes via repository layer.
- Apply writes audit log entry.
- Apply stores rollback payload.
- Rollback restores original event state when no external conflict exists.
- Rollback blocks with clear explanation when external ETag changed.
- Tests prove stale proposals cannot apply.
- Tests prove no overlapping focus block can be applied over fixed events.
- Todoist update and message draft are represented as placeholder diff types but not executed.

---

## 16. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| Rollback corrupts calendar after external edit | Critical | ETag-at-apply checks before rollback |
| Apply partially succeeds | High | DB transactions for event changes + audit |
| Diff schema too narrow for future LLMs | Medium | Include add/move/delete/update/placeholders now |
| Audit log stores too much private detail | Medium | Store rollback payload internally; redact API responses |
| Deterministic slot finder feels weak | Low | PRP goal is safety skeleton, not magic |
| Recurrence edits explode complexity | High | Do not edit complex recurrence in MVP |

---

## 17. Open questions

- Should rollback payload store raw ICS encrypted at rest, or rely on DB file ownership for MVP?
- Should proposal expiration be 15 minutes or shorter?
- Should destructive changes require a separate confirm token now, or only when UI implements destructive actions?
- Should focus blocks live in a dedicated calendar or be tagged events in default calendar?
- Should audit log expose raw before/after to local admin only, or never by default?

---

## 18. Suggested GitHub issue breakdown

1. `feat(planner): add proposal and diff domain types`
2. `feat(planner): implement constraint engine`
3. `feat(planner): implement deterministic focus slot finder`
4. `feat(data): persist plan proposals and rollback payloads`
5. `feat(api): add daily proposal and proposal retrieval endpoints`
6. `feat(api): add apply endpoint with revalidation`
7. `feat(api): add rollback endpoint with ETag conflict checks`
8. `feat(api): add audit log endpoints`
9. `test(planner): cover constraints and stale proposal behavior`
10. `docs(architecture): document planning safety flow`
