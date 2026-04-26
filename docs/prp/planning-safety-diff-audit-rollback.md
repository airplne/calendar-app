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
- Ensure invalid, expired, stale, or unsupported proposals cannot be applied.
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
- No implementation of `todoist_update` or `message_draft` apply behavior in MVP.

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
    CanApply         bool
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
    Type                ViolationType
    Severity            string
    Message             string
    ProposedChangeID    string
    ConflictingEntityID string
    ConflictingUID      string
}

type RollbackPayload struct {
    ProposalID          string
    AuditLogEntryID     int64
    ChangesToUndo       []RollbackChange
    AffectedRefsAtApply []AffectedEntityRef
    CreatedAt           time.Time
}
```

### FR2 - Proposal status lifecycle

Canonical `ProposalStatus` values:

- `draft`: proposal is being assembled and is not visible/actionable.
- `validated`: proposal passed validation and may be actionable if `can_apply=true`.
- `invalid`: proposal failed validation before display and cannot be applied.
- `expired`: proposal can no longer be applied because its time-based expiration passed.
- `stale`: proposal can no longer be applied because affected calendar state changed since proposal generation, such as changed ETags or deleted affected events.
- `applied`: proposal was successfully applied.
- `rejected`: user rejected the proposal.
- `apply_failed`: apply was attempted but failed for a non-stale reason.
- `rolled_back`: applied proposal was successfully rolled back.
- `rollback_blocked`: rollback was attempted but blocked, usually because affected event ETags changed after apply.

Status transition rules:

- Validation failed before display -> `invalid`.
- Expired due to time -> `expired`.
- Calendar changed / ETag mismatch / affected event deleted before apply -> `stale`.
- Apply attempted but failed for non-stale reason -> `apply_failed`.
- Rollback conflict after external event change -> `rollback_blocked`.

Suggested user-facing stale copy:

```text
Calendar changed. Regenerate before applying.
```

### FR3 - Diff format and supported change types

Every proposal must expose machine-readable and human-readable diffs.

Schema change types:

- `add`
- `move`
- `delete`
- `update`
- `todoist_update` placeholder
- `message_draft` placeholder

MVP support:

- MVP apply supports only `add` of a calendar event for a protected focus block.
- `move`, `delete`, and `update` are schema-ready but not MVP-supported for apply unless a later implementation issue explicitly adds support and tests.
- `todoist_update` and `message_draft` are placeholders for future compatibility and are not MVP-supported.

Unsupported placeholder or schema-only change types must produce:

- `can_apply=false`
- `validation_result.valid=false`
- `error_code=unsupported_change_type`

### FR4 - Constraint engine

Implement constraints:

1. `TimeConflictConstraint`: no proposed add/move may overlap fixed confirmed events.
2. `TimeWindowConstraint`: proposed focus block must fit configured working hours; default 09:00-17:00 local time.
3. `ImmovableEventConstraint`: proposed add cannot overlap an immovable event; future move/delete/update cannot affect immovable events.
4. `FocusBlockProtectionConstraint`: proposed changes must not weaken or overlap protected focus blocks.
5. `MinimumFocusDurationConstraint`: focus block must meet default minimum duration, initially 60 minutes unless preference exists.
6. `SupportedChangeTypeConstraint`: unsupported change types make the proposal invalid and non-actionable.

Validation must happen before proposal response, immediately before apply, and immediately before rollback where relevant.

Invalid proposal behavior:

- API may return invalid proposal with violations for debugging, but `can_apply=false`.
- UI must not show actionable Apply for invalid proposal.
- Apply endpoint must reject invalid proposals regardless of UI behavior.

### FR5 - Proposal storage

Persist proposals long enough to apply or expire.

Requirements:

- Store proposal as JSON and normalized metadata.
- Default expiration: 15 minutes.
- Store canonical status from FR2.
- Store affected event refs: `calendar_id`, `uid`, `etag_at_generation`, `start_time`, `end_time`.
- Expired proposals cannot apply and transition to `expired`.
- If affected ETags differ at apply time, proposal transitions to `stale`.

### FR6 - Deterministic daily proposal

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

### FR7 - Proposal retrieval

Implement:

```http
GET /api/v1/plan/proposals/{proposal_id}
```

Returns stored proposal, current status, validation result, expiration, and `can_apply` state.

### FR8 - Apply endpoint

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
3. Reject if expired; transition proposal to `expired`.
4. Reject if already applied/rolled back/rejected.
5. Reject if `can_apply=false`.
6. Re-read affected events.
7. Compare current ETags to stored `etag_at_generation`.
8. If affected calendar state changed, transition proposal to `stale` and return `error_code=stale_proposal`.
9. Re-run constraints on current calendar state.
10. If validation fails, transition proposal to `apply_failed` or `stale` depending on cause.
11. Begin transaction.
12. Apply supported changes through repository layer.
13. Store rollback payload.
14. Write audit log entry.
15. Mark proposal `applied`.
16. Commit transaction.
17. Return applied summary.

Avoid partial apply:

- All supported event changes must be applied in one transaction.
- If any change fails, rollback the transaction.
- Future external integrations should use a separate compensation design; out of scope here.

Required apply success response:

```json
{
  "status": "applied",
  "proposal_id": "prop_123",
  "audit_log_entry_id": 42,
  "rollback_available": true,
  "can_apply": false,
  "summary": "Focus block added",
  "applied_changes": [
    {
      "change_id": "change_1",
      "type": "add",
      "target_type": "calendar_event",
      "title": "Deep Work",
      "start": "2026-04-28T09:30:00-04:00",
      "end": "2026-04-28T11:30:00-04:00"
    }
  ]
}
```

Required stale failure response:

```json
{
  "status": "stale",
  "proposal_id": "prop_123",
  "error_code": "stale_proposal",
  "message": "Calendar changed. Regenerate before applying.",
  "can_apply": false,
  "rollback_available": false,
  "summary": "Proposal is stale",
  "applied_changes": []
}
```

Required apply response fields:

- `status`.
- `proposal_id`.
- `audit_log_entry_id` when apply succeeds.
- `rollback_available`.
- `summary`.
- `applied_changes`.
- `can_apply`.
- `error_code` on failure.
- `message` on failure.

### FR9 - Rollback endpoint

Implement:

```http
POST /api/v1/plan/rollback
```

Request options:

```json
{ "proposal_id": "prop_123" }
```

or:

```json
{ "audit_log_entry_id": 42 }
```

Rollback behavior:

1. Load applied audit entry and rollback payload.
2. Re-read all affected current events.
3. Compare current ETags to `AffectedRefsAtApply`.
4. If any ETag changed externally, block rollback with `error_code=rollback_conflict`.
5. Explain which event changed and why rollback is blocked.
6. If safe, begin transaction.
7. Undo applied changes. MVP supports undo for added focus-block event by deleting that event.
8. Write rollback audit log entry.
9. Mark original proposal `rolled_back`.
10. Commit transaction.
11. Return rollback summary.

### FR10 - Audit log and rollback payload privacy

Extend existing `audit_log` model if needed.

Audit entries must include user ID, action, proposal ID, human summary, machine-readable changes, rollback payload reference, result, created timestamp, and redacted errors.

Expose:

```http
GET /api/v1/audit-log
GET /api/v1/audit-log/{entry_id}
```

MVP rollback payload storage decision:

Rollback payloads may store full event snapshots internally in SQLite for MVP because Calendar-app is self-hosted and rollback requires faithful restoration.

However:

- rollback payloads must not be returned by default API responses;
- rollback payloads must not be included in default debug bundles;
- audit log views must show redacted summaries by default;
- future encryption-at-rest can be considered post-MVP.

### FR11 - Rejection endpoint

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

Supported reason codes: `bad_time`, `too_long`, `too_short`, `conflicts_with_preference`, `not_enough_context`, `other`.

### FR12 - Error codes

Use these error codes consistently in API responses, tests, and UX mapping:

- `proposal_expired`
- `stale_proposal`
- `validation_failed`
- `unsupported_change_type`
- `rollback_conflict`
- `sync_unhealthy`
- `apply_failed`
- `rollback_failed`

---

## 7. Non-functional requirements

- Determinism: same calendar state + same preferences = same mock proposal.
- Safety: apply always revalidates.
- Atomicity: calendar mutations and audit write happen in the same DB transaction where possible.
- Privacy: audit entries and diffs avoid raw ICS in default API responses; full event snapshots may be stored internally for rollback but not exposed by default.
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
  supported_change_type.go
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

### Validation flow

```text
Generate candidate
-> Build CalendarState from EventRepo
-> Validate constraints including supported change types
-> Store proposal with validation result and affected refs
-> Return proposal with can_apply = validation.valid && status == validated && not expired
```

### Apply flow

```text
Load proposal
-> Check status/expiration/can_apply
-> Re-read affected events
-> Check generation ETags
-> If changed, mark stale and return stale_proposal
-> Rebuild CalendarState
-> Revalidate constraints
-> Begin transaction
-> Apply event changes
-> Store rollback payload
-> Write audit log
-> Mark proposal applied
-> Commit
-> Return audit_log_entry_id and rollback_available
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
server/internal/planner/supported_change_type.go
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

Common stale error response:

```json
{
  "status": "stale",
  "proposal_id": "prop_123",
  "error_code": "stale_proposal",
  "message": "Calendar changed. Regenerate before applying.",
  "can_apply": false
}
```

Unsupported change type response:

```json
{
  "status": "invalid",
  "proposal_id": "prop_123",
  "error_code": "unsupported_change_type",
  "message": "This proposal includes a change type that is not supported by MVP apply.",
  "can_apply": false
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
  status TEXT NOT NULL CHECK (status IN (
    'draft', 'validated', 'invalid', 'expired', 'stale', 'applied',
    'rejected', 'apply_failed', 'rolled_back', 'rollback_blocked'
  )),
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

Backend-focused PRP, but API must support future UI:

- `can_apply` boolean.
- `validation_result.valid`.
- `validation_result.violations`.
- grouped diffs by change type.
- `audit_log_entry_id` on successful apply.
- `rollback_available` on apply response.
- rollback conflict explanation.
- stale proposal copy: `Calendar changed. Regenerate before applying.`

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
- `TestSupportedChangeTypeConstraint_BlocksUnsupportedTypes`.

Proposal tests:

- `TestPlanDaily_ReturnsDeterministicProposal`.
- `TestPlanDaily_ReturnsInvalidWhenNoSafeSlot`.
- `TestProposalExpires`.
- `TestStoredProposalIncludesAffectedETags`.
- `TestUnsupportedChangeTypeReturnsCanApplyFalse`.

Apply tests:

- `TestApply_RevalidatesBeforeWrite`.
- `TestApply_RejectsInvalidProposal`.
- `TestApply_RejectsExpiredProposalAndMarksExpired`.
- `TestApply_RejectsStaleETagAndMarksStale`.
- `TestApply_RejectsUnsupportedChangeType`.
- `TestApply_WritesEventViaRepository`.
- `TestApply_WritesAuditLog`.
- `TestApply_ReturnsAuditLogEntryID`.
- `TestApply_ReturnsRollbackAvailable`.
- `TestApply_StoresRollbackPayload`.
- `TestApply_TransactionRollbackOnFailure`.

Rollback tests:

- `TestRollback_DeletesAddedFocusBlock`.
- `TestRollback_BlocksWhenExternalETagChanged`.
- `TestRollback_WritesAuditLogEntry`.
- `TestRollback_CannotRollbackTwice`.

API redaction tests:

- Audit log API responses do not include raw ICS by default.
- Audit log API responses show human-readable summaries and redacted metadata only.
- Rollback payload event snapshots remain internal unless explicitly requested through a future authenticated diagnostic path.

API tests:

- `TestPostPlanDaily`.
- `TestGetProposal`.
- `TestPostApplySuccessResponseShape`.
- `TestPostApplyStaleFailureResponseShape`.
- `TestPostRollback`.
- `TestGetAuditLog`.
- `TestRejectProposal`.

---

## 14. Manual validation steps

1. Run `cd server && go test ./...`.
2. Start server.
3. Seed calendar with fixed events at 09:00-09:30 and 12:00-13:00.
4. Call `POST /api/v1/plan/daily` with today's date.
5. Confirm response proposes a non-overlapping focus block with `can_apply=true`.
6. Apply proposal.
7. Confirm response includes `audit_log_entry_id`, `rollback_available`, `summary`, and `applied_changes`.
8. Confirm event exists in `events`.
9. Confirm audit entry exists.
10. Rollback proposal using either `proposal_id` or `audit_log_entry_id`.
11. Confirm event removed/restored.
12. Generate another proposal.
13. Modify affected event externally.
14. Attempt apply.
15. Confirm proposal transitions to `stale` and response says `Calendar changed. Regenerate before applying.`
16. Apply a proposal, externally modify focus event, then attempt rollback.
17. Confirm rollback conflict error.

---

## 15. Acceptance criteria

- `go test ./...` passes.
- Constraint tests cover overlap, allowed windows, immovable events, protected focus blocks, and unsupported change types.
- `/api/v1/plan/daily` can return a deterministic/mock validated proposal.
- Proposal includes machine-readable changes and human-readable diff summaries.
- Invalid proposals return violations and cannot be applied.
- Unsupported placeholder change types return `can_apply=false`, `validation_result.valid=false`, and `error_code=unsupported_change_type`.
- `/api/v1/plan/apply` revalidates before write.
- Apply checks affected event ETags against proposal generation refs.
- Stale ETag apply attempts transition the proposal to `stale`.
- Stale proposals cannot be applied.
- Stale proposals return a user-facing error instructing the user to regenerate the proposal.
- Apply success response includes `status`, `proposal_id`, `audit_log_entry_id`, `rollback_available`, `summary`, `applied_changes`, and `can_apply`.
- Apply failure response includes `status`, `proposal_id`, `error_code`, `message`, and `can_apply=false`.
- Apply writes event changes via repository layer.
- Apply writes audit log entry.
- Apply stores rollback payload.
- Rollback restores original event state when no external conflict exists.
- Rollback blocks with clear explanation when external ETag changed.
- Tests prove stale proposals cannot apply.
- Tests prove no overlapping focus block can be applied over fixed events.
- Todoist update and message draft are represented as placeholder diff types but not executed.
- Audit log API responses do not expose raw ICS by default.

---

## 16. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| Rollback corrupts calendar after external edit | Critical | ETag-at-apply checks before rollback |
| Apply partially succeeds | High | DB transactions for event changes + audit |
| Diff schema too narrow for future LLMs | Medium | Include add/move/delete/update/placeholders now, but block unsupported apply |
| Audit log stores too much private detail | Medium | Store rollback payload internally; redact API responses |
| Rollback event snapshots leak through debug/audit APIs | High | Redaction tests; exclude rollback payloads from default debug bundle |
| Deterministic slot finder feels weak | Low | PRP goal is safety skeleton, not magic |
| Recurrence edits explode complexity | High | Do not edit complex recurrence in MVP |

---

## 17. Open questions

- Should proposal expiration be 15 minutes or shorter?
- Should destructive changes require a separate confirm token now, or only when UI implements destructive actions?
- Should focus blocks live in a dedicated calendar or be tagged events in default calendar?
- Should a future authenticated diagnostic path expose rollback event snapshots, or should they remain internal only?
- Should future encryption-at-rest be added for rollback payloads post-MVP?

---

## 18. Suggested GitHub issue breakdown

1. `feat(planner): add proposal and diff domain types`
2. `feat(planner): implement canonical proposal statuses including stale`
3. `feat(planner): implement constraint engine`
4. `feat(planner): implement supported change type validation`
5. `feat(planner): implement deterministic focus slot finder`
6. `feat(data): persist plan proposals and rollback payloads`
7. `feat(api): add daily proposal and proposal retrieval endpoints`
8. `feat(api): add apply endpoint with revalidation and response contract`
9. `feat(api): add rollback endpoint with ETag conflict checks`
10. `feat(api): add audit log endpoints with redacted responses`
11. `test(planner): cover constraints, unsupported changes, and stale proposal behavior`
12. `docs(architecture): document planning safety flow`
