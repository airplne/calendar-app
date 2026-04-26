# PRP: First Aha UX - Agenda, Focus Proposal, Diff Review, and Undo

**Type:** Implementation PRP / Product Requirements Prompt  
**Target Team:** Frontend + Backend API + UX QA  
**Status:** Proposed  
**Priority:** MVP user-visible trust loop

---

## 1. Context

The first visible Calendar-app aha should be small, safe, and concrete: Calendar-app finds one open slot, proposes one protected focus block, validates it, shows the diff, applies after confirmation, logs it, and can undo it.

This PRP turns the backend planning safety skeleton into an agenda-first user experience. It depends on:

- Sync Health status and green-sync validation from the CalDAV trust PRP.
- Proposal/diff/apply/audit/rollback APIs from the planning safety PRP.
- PRP 2's apply response contract, specifically `audit_log_entry_id`, `rollback_available`, `can_apply`, and `applied_changes`.

The experience should work without a real LLM and without Todoist. The point is to make the trust promise visible.

---

## 2. Problem statement

Users need to experience Calendar-app's trust promise in the UI, not just in backend safety code. A chat-first or full-day planning experience is too broad for the first demo. The MVP should show one validated focus block proposal in an agenda-first dashboard, with visible reasoning, diff, apply, audit, and undo.

---

## 3. Goals

- Create Today dashboard with agenda and sync health status.
- Show one recommended focus block proposal after an explicit user action.
- Explain why the slot was chosen.
- Show constraints checked and validation status.
- Show a clear diff before apply.
- Enable Apply only when proposal, validation, and sync-health gates pass.
- Show success banner with Undo and View audit log using the apply response contract.
- Capture rejection reason.
- Handle no-slot, unhealthy sync, invalid proposal, stale proposal, apply failure, and rollback failure states.

---

## 4. Non-goals

- No chat-first interface.
- No multiple alternatives required.
- No real LLM required.
- No Todoist dependency required.
- No complex drag/drop calendar editing.
- No destructive move/delete flows in first aha, except placeholders.
- No mobile-native app.
- No ambient automatic replanning.
- No automatic proposal generation on page load for MVP.

---

## 5. MVP interaction decisions

### Explicit click-to-generate

MVP uses an explicit `Find focus slot` / `Generate proposal` action.

The first aha should not auto-generate and present changes before the user asks. This keeps the experience trust-building and avoids surprising users.

### Apply enablement rule

When `sync_status != healthy` or green-sync validation is incomplete, proposal generation may be shown for preview only, but Apply must be disabled with a link to Sync Health/onboarding.

Apply may be enabled only when all of the following are true:

- `can_apply === true`
- `validation_result.valid === true`
- `sync_status === healthy`
- green-sync validation is complete

The frontend must not infer rollback availability from local state. It must use `rollback_available` from the apply response.

---

## 6. User stories

1. As a user, I can open Today and see my agenda plus sync health.
2. As a user, I can explicitly click to find one suggested protected focus block that fits my day.
3. As a user, I can understand why the slot was chosen and what constraints passed before applying.
4. As a user, I can open a diff and explain what will change.
5. As a user, I can apply the proposal only after confirmation and only when sync health is healthy.
6. As a user, I can undo the applied focus block if the apply response says rollback is available and nothing changed externally.
7. As a user, I can use View audit log to open the audit entry returned by apply.
8. As a user, I can reject a bad proposal with a structured reason.
9. As a user, I understand what to do when sync is unhealthy, no slot exists, proposal is invalid, proposal is stale, apply fails, or rollback is blocked.

---

## 7. Functional requirements

### FR1 - Daily dashboard

Add route:

```text
/today
```

Dashboard sections:

1. Header
   - Date.
   - Sync status.
   - Calendar source: Calendar-app CalDAV.
   - Planner state: deterministic planner / LLM not configured / configured if visible.
2. Agenda
   - Today's events sorted by time.
   - Focus blocks visually distinguished.
   - Current time marker if simple.
3. Availability summary
   - Total open working time.
   - Usable focus time.
   - Longest open slot.
4. Focus protection summary
   - Protected focus blocks today.
   - Interruptions detected, if any.
5. Recommended plan card
   - One proposal after explicit user action.
   - Explanation.
   - Validation.
   - Diff summary.
   - Actions.
   - Generated/expiration timestamp.

### FR2 - Today API dependency

Use these endpoints:

```http
GET  /api/v1/today?date=YYYY-MM-DD
POST /api/v1/plan/daily
GET  /api/v1/plan/proposals/{proposal_id}
POST /api/v1/plan/apply
POST /api/v1/plan/rollback
POST /api/v1/plan/proposals/{proposal_id}/reject
GET  /api/v1/sync-health
GET  /api/v1/audit-log/{entry_id}
```

If `GET /api/v1/today` does not exist yet, add it in this PRP.

Example response:

```json
{
  "date": "2026-04-25",
  "sync_status": "healthy",
  "green_sync_completed": true,
  "calendar_source": "Calendar-app CalDAV",
  "events": [
    {
      "uid": "evt_1",
      "summary": "Standup",
      "start_time": "2026-04-25T08:30:00-04:00",
      "end_time": "2026-04-25T09:00:00-04:00",
      "kind": "meeting",
      "protection_level": "normal"
    }
  ],
  "availability": {
    "usable_focus_time_minutes": 150,
    "longest_open_slot_minutes": 120
  },
  "focus_summary": {
    "protected_blocks_count": 0,
    "interrupted_blocks_count": 0
  }
}
```

### FR3 - First proposal card

Before generation, show an explicit action:

```text
Ready to protect focus time?

[Find focus slot]
```

Proposal card must show:

```text
Suggested protection:
Add "Deep Work" from 9:30-11:30

Using deterministic planner. No LLM data sent.

Generated 2 minutes ago · Expires in 8 minutes

Why:
- Longest open morning slot
- No fixed conflicts
- Inside configured working hours
- Preserves lunch and existing meetings

Validation:
Passed

Changes:
+ Add focus block: Deep Work, 9:30-11:30

[Show diff] [Apply] [Reject]
```

Requirements:

- Use deterministic proposal from planning safety PRP.
- Show only one recommended proposal.
- Do not show or enable Apply unless the Apply enablement rule passes.
- If proposal expires, show `Proposal expired. Regenerate before applying.`
- If proposal becomes stale, affected event ETags changed, or sync state changed after generation, show `Calendar changed. Regenerate before applying.`
- If sync status is unhealthy, show warning before proposal generation: `Calendar sync is unhealthy. You can preview a proposal, but Apply is disabled until sync is healthy.`
- If no LLM key exists, do not block first proposal; show `Using deterministic planner. No LLM data sent.`

### FR4 - Explanation panel

Show `Why this slot?` with:

- Slot selection reason.
- Constraints checked.
- Tradeoffs/assumptions.
- Existing events preserved.
- Working hours used.
- Focus duration used.

Example:

```text
Why this slot?
This is the longest open morning slot inside your working hours. It does not overlap fixed events and leaves lunch untouched.
```

### FR5 - Validation status

Show:

- Passed / Failed / Warning.
- Checked timestamp.
- Constraint list:
  - Time conflicts.
  - Working hours.
  - Immovable events.
  - Protected focus blocks.
  - Minimum focus duration.

If invalid:

- Disable Apply.
- Show each violation.
- Provide `Regenerate` if endpoint is available.
- Provide `Reject` with reason.

### FR6 - Diff preview

Add a modal/drawer or inline expandable panel.

Grouped sections:

- Add.
- Move.
- Delete.
- Update.
- Todoist update placeholder.
- Message draft placeholder.

MVP content likely only:

```text
Add
+ Focus block: Deep Work
  Time: 9:30-11:30
  Calendar: Default
  Protection: Protected
```

Technical details expandable:

- Proposal ID.
- Change ID.
- UID.
- Calendar ID.
- Validation checked timestamp.
- ETag refs count.
- Generated at.
- Expires at.

### FR7 - Apply confirmation

Apply behavior:

- Button label: `Apply focus block`.
- Apply is disabled unless the Apply enablement rule passes.
- Disabled Apply state links the user to Sync Health or onboarding remediation.
- On click, show lightweight confirmation if only adding a focus block.
- Strong confirmation is reserved for destructive changes.
- Loading state disables buttons and shows `Revalidating and applying...`.

Success response handling:

- Use `audit_log_entry_id` from the apply response for `View audit log`.
- Show Undo only when `rollback_available === true` in the apply response.
- Use `applied_changes` from the apply response to update/refetch the agenda.
- Do not infer rollback availability from local state.

Success banner:

```text
Focus block added.

[Undo] [View audit log]
```

Failure handling:

- Stale proposal: `Calendar changed. Regenerate before applying.`
- Expired proposal: `Proposal expired. Regenerate before applying.`
- Validation failed: show violations and no Apply.
- ETag changed: use stale proposal copy.
- Server error: show retry-safe error.
- Sync unhealthy: link to Sync Health.

### FR8 - Undo after apply

Undo action:

- Calls `/api/v1/plan/rollback` with `proposal_id` or `audit_log_entry_id` from the apply result.
- Show Undo only when `rollback_available === true`.

On success:

- Remove focus block from agenda after server confirms rollback.
- Update proposal state to rolled back.
- Show:

```text
Focus block removed. Your calendar is back to its previous state.
```

On blocked rollback:

- Show:

```text
Undo blocked because this event changed in another calendar client after apply. Review the audit log for details.
```

- Provide link to audit log entry using the known `audit_log_entry_id`.

### FR9 - Reject flow

Reject button opens a small dialog.

Structured reasons:

- Bad time.
- Too long.
- Too short.
- Conflicts with preference.
- Not enough context.
- Other.

Free text optional.

POST rejection to backend.

After reject:

- Hide Apply.
- Show `Proposal rejected`.
- Optional `Generate another` placeholder or enabled if endpoint supports it.

### FR10 - Empty and error states

Required states:

1. No open focus slot
   - `No safe focus slot found today.`
   - Explain top blockers.
2. Calendar not synced
   - `Connect a CalDAV client and reach green sync first.`
   - Link to onboarding.
3. Sync unhealthy
   - `Sync health needs attention before applying plans.`
   - Link to Sync Health.
   - Apply disabled.
4. Proposal invalid
   - Show violations; no Apply.
5. Proposal stale
   - `Calendar changed. Regenerate before applying.`
   - No Apply.
6. Apply failed
   - Show error and next step.
7. Rollback failed
   - Show reason and link to audit log.
8. LLM not configured
   - `Using deterministic planner. No LLM data sent.`
9. Proposal expired
   - `Proposal expired. Regenerate before applying.`
   - Show regenerate action.

### FR11 - Proposal expiration display

Proposal card should show when it was generated and/or when it expires.

Example:

```text
Generated 2 minutes ago · Expires in 8 minutes
```

If expired:

```text
Proposal expired. Regenerate before applying.
```

---

## 8. Non-functional requirements

- Trust clarity: user must be able to identify what will change before applying.
- Accessibility: keyboard-accessible modal/drawer, focus trap, ARIA labels, WCAG 2.1 AA contrast.
- Responsiveness: desktop-first, responsive enough for mobile review/apply.
- Performance: Today dashboard loads under 1s locally for typical single-user data.
- No silent writes: every apply originates from explicit user action.
- No surprise proposals: first aha requires explicit click-to-generate.
- Resilience: UI handles API errors without losing local state.
- Privacy: do not expose raw ICS in technical details.
- State correctness: agenda updates after apply/undo via refetch or local mutation confirmed by server response.

---

## 9. Technical approach

### Frontend architecture

Components:

```text
TodayPage
  TodayHeader
  SyncStatusBadge
  AgendaTimeline
  AvailabilitySummary
  FocusProtectionSummary
  GenerateProposalButton
  ProposalCard
  ProposalExplanation
  ValidationBadge
  ConstraintChecklist
  DiffPreviewDrawer
  ApplyConfirmation
  UndoBanner
  RejectProposalDialog
  EmptyStatePanel
  ErrorStatePanel
```

State handling:

- TanStack Query:
  - `useToday(date)`.
  - `useSyncHealth()`.
  - `useDailyProposal(date)`.
  - `useProposal(proposalId)`.
  - `useApplyProposal()`.
  - `useRollbackProposal()`.
  - `useRejectProposal()`.
- Zustand only if needed for UI drawer/dialog state.

Query invalidation:

- After apply: invalidate `today`, `proposal`, and `audit-log`.
- After rollback: invalidate `today`, `proposal`, and `audit-log`.

Apply button derived state:

```ts
const applyEnabled =
  proposal.can_apply === true &&
  proposal.validation_result.valid === true &&
  syncHealth.status === "healthy" &&
  syncHealth.green_sync_completed === true;
```

### Backend additions

Add `GET /api/v1/today` if missing:

- Uses EventRepo to list events by date.
- Uses SyncHealthService for status summary.
- Computes availability from working hours and events.
- Computes focus summary.

---

## 10. Files likely to change

Frontend:

```text
web/src/pages/TodayPage.tsx
web/src/components/today/TodayHeader.tsx
web/src/components/today/AgendaTimeline.tsx
web/src/components/today/AvailabilitySummary.tsx
web/src/components/today/FocusProtectionSummary.tsx
web/src/components/plan/GenerateProposalButton.tsx
web/src/components/plan/ProposalCard.tsx
web/src/components/plan/ProposalExplanation.tsx
web/src/components/plan/ValidationBadge.tsx
web/src/components/plan/ConstraintChecklist.tsx
web/src/components/plan/DiffPreviewDrawer.tsx
web/src/components/plan/RejectProposalDialog.tsx
web/src/components/plan/UndoBanner.tsx
web/src/components/common/EmptyStatePanel.tsx
web/src/components/common/ErrorStatePanel.tsx
web/src/lib/api.ts
web/src/lib/types.ts
web/src/App.tsx
```

Backend:

```text
server/internal/api/today.go
server/internal/api/plan.go
server/internal/planner/slot_finder.go
server/internal/planner/proposal_service.go
server/internal/api/today_test.go
```

Tests:

```text
web/src/pages/TodayPage.test.tsx
web/src/components/plan/*.test.tsx
server/internal/api/today_test.go
```

Docs:

```text
docs/ux/first-aha-flow.md
```

---

## 11. API changes

Add if missing:

```http
GET /api/v1/today?date=YYYY-MM-DD
```

Use from dependency PRPs:

```http
POST /api/v1/plan/daily
GET  /api/v1/plan/proposals/{proposal_id}
POST /api/v1/plan/apply
POST /api/v1/plan/rollback
POST /api/v1/plan/proposals/{proposal_id}/reject
GET  /api/v1/sync-health
GET  /api/v1/audit-log/{entry_id}
```

Apply response requirements inherited from PRP 2:

- `status`.
- `proposal_id`.
- `audit_log_entry_id` when apply succeeds.
- `rollback_available`.
- `summary`.
- `applied_changes`.
- `can_apply`.
- `error_code` on failure.
- `message` on failure.

---

## 12. Database/schema changes

No required new schema if the CalDAV trust and planning safety PRPs are complete.

Optional if rejection reason is not included in the planning safety PRP:

```sql
CREATE TABLE proposal_rejections (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  proposal_id TEXT NOT NULL,
  reason_code TEXT NOT NULL,
  free_text TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (proposal_id) REFERENCES plan_proposals(id) ON DELETE CASCADE
);
```

---

## 13. UX changes

Primary UX route: `/today`.

Screen-level requirements:

### Today dashboard

Top row:

```text
Today: Sat Apr 25
Sync: Healthy
Calendar source: Calendar-app CalDAV
Planner: Deterministic
```

Left/main:

- Agenda timeline.

Right/side or top card:

- Find focus slot CTA.
- Suggested protection card after generation.

Below or side:

- Availability summary.
- Focus protection summary.

### Proposal card copy

Use plain language:

- `Find focus slot`
- `Suggested protection`
- `Using deterministic planner. No LLM data sent.`
- `Generated 2 minutes ago · Expires in 8 minutes`
- `Why this slot`
- `Validation passed`
- `What will change`
- `Apply focus block`
- `Reject`

Avoid:

- `AI optimized your schedule`
- `Autopilot`
- `We changed your calendar`

### Stale / expired recovery copy

Use:

```text
Calendar changed. Regenerate before applying.
```

when:

- proposal is stale;
- affected event ETags changed;
- sync state changed after proposal generation.

Use:

```text
Proposal expired. Regenerate before applying.
```

when the proposal expiration time has passed.

### Accessibility requirements

- All buttons keyboard reachable.
- Disabled Apply state remains understandable and includes remediation link.
- Diff drawer closes with Escape.
- Focus returns to opener after drawer closes.
- Validation status announced to screen readers.
- Loading states use `aria-busy`.
- Error banners use `role="alert"`.

---

## 14. Test plan

Frontend unit/component tests:

- Today dashboard renders agenda.
- Sync status badge renders healthy/warning/critical/unknown.
- Proposal generation requires explicit click.
- Proposal card renders explanation, deterministic planner copy, generated timestamp, expiration, and validation.
- Apply hidden/disabled for invalid proposal.
- Apply disabled when Sync Health is warning, critical, unknown, or green-sync validation is incomplete.
- Disabled Apply state links the user to Sync Health or onboarding remediation.
- Apply enabled only when `can_apply`, validation, sync health, and green-sync gates pass.
- Diff preview opens and groups changes.
- Apply success uses `audit_log_entry_id` for View audit log.
- Undo is shown only when `rollback_available === true`.
- Undo success removes banner/refetches agenda.
- Rollback blocked shows clear error.
- Reject dialog submits reason.
- No open slot empty state renders.
- Stale proposal copy renders.
- Sync unhealthy warning links to Sync Health.

Frontend integration tests:

- Mock full happy path: load today, click Find focus slot, generate proposal, open diff, apply, see success, view audit log link, undo.
- Mock stale proposal apply failure.
- Mock invalid proposal.
- Mock unhealthy sync state and confirm Apply disabled.

Backend tests:

- `GET /api/v1/today` returns events and availability.
- Availability excludes fixed events.
- Availability computes longest open slot.
- Today endpoint includes sync status and green-sync completion.

UX test:

- Give user a proposal screen.
- Ask: `What will change if you click Apply?`
- Pass if user says it will add a Deep Work focus block from X to Y and does not think existing events will move/delete.

---

## 15. Manual validation steps

1. Complete green sync from the CalDAV trust PRP.
2. Seed today calendar:
   - Standup 08:30-09:00.
   - Lunch 12:00-13:00.
   - Meeting 14:00-15:00.
3. Open `/today`.
4. Confirm agenda appears.
5. Confirm Sync Health status appears.
6. Click `Find focus slot`.
7. Confirm proposal suggests one safe focus block.
8. Confirm deterministic planner copy and proposal expiration display appear.
9. Open diff.
10. Confirm diff says only one event will be added.
11. Apply.
12. Confirm focus block appears in agenda and external CalDAV client.
13. Confirm success banner uses `audit_log_entry_id` for View audit log.
14. Confirm Undo appears only when `rollback_available=true`.
15. Click Undo.
16. Confirm focus block disappears from agenda and external CalDAV client.
17. Repeat with intentionally invalid proposal fixture; confirm no Apply.
18. Repeat with unhealthy sync state; confirm proposal can preview but Apply is disabled and links to remediation.
19. Repeat with external edit after apply; confirm Undo blocked clearly.

---

## 16. Acceptance criteria

- User can view Today agenda.
- User can see Sync Health status.
- User can explicitly generate or view one proposed focus block via `Find focus slot` / `Generate proposal`.
- Proposal includes explanation and validation status.
- Proposal includes deterministic planner copy: `Using deterministic planner. No LLM data sent.`
- Proposal shows generated/expiration timing.
- Proposal shows constraints checked.
- User can open diff preview.
- Apply is disabled when Sync Health is warning, critical, unknown, or green-sync validation is incomplete.
- Disabled Apply state links the user to Sync Health or onboarding remediation.
- Tests cover healthy and unhealthy sync states.
- User can apply the proposal only when `can_apply === true`, `validation_result.valid === true`, `sync_status === healthy`, and green-sync validation is complete.
- Applied focus block appears on the calendar.
- Audit log entry is created.
- Success banner includes View audit log using `audit_log_entry_id` from apply response.
- Success banner includes Undo only when `rollback_available === true`.
- User can undo the apply if no external conflict exists.
- Undo blocked state is clear when external ETag changed.
- Invalid proposal does not show actionable Apply.
- Stale proposal shows `Calendar changed. Regenerate before applying.`
- Expired proposal shows `Proposal expired. Regenerate before applying.`
- Reject flow captures structured reason and optional free text.
- No LLM key is required for first aha.
- UX test passes: user can explain what will change before applying.
- `go test ./...` passes.
- Frontend build/test passes using project-standard commands.

---

## 17. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| First aha feels too small | Medium | Make trust visible: why, validation, diff, undo |
| UI overexplains | Medium | Default concise card; technical details expandable |
| Apply race causes failure | Medium | Treat stale proposal as expected state with regenerate CTA |
| Sync unhealthy still allows writes | High | Hard disable Apply unless sync and green-sync gates pass |
| Undo blocked surprises user | Medium | Explain external changes and link to audit log |
| Agenda UI becomes scope creep | High | No drag/drop; no week view; Today only |
| No-slot state feels like failure | Low | Explain blockers and suggest retry/edit preferences later |

---

## 18. Open questions

- Should the first focus block default to 90 minutes or 2 hours?
- Should `Deep Work` title be configurable before apply or fixed for MVP?
- Should focus blocks use a dedicated calendar if one exists?
- Should success banner persist until dismissed or disappear after navigation?

Resolved for MVP:

- Proposal generation is explicit click-to-generate, not automatic on page load.

---

## 19. Suggested GitHub issue breakdown

1. `feat(api): add today dashboard endpoint`
2. `feat(web): add Today dashboard route`
3. `feat(web): add agenda timeline component`
4. `feat(web): add sync status and availability summaries`
5. `feat(web): add click-to-generate proposal card with explanation and validation`
6. `feat(web): add diff preview drawer`
7. `feat(web): add apply proposal flow with sync-health gate`
8. `feat(web): add undo rollback banner using apply response contract`
9. `feat(web): add rejection reason dialog`
10. `test(web): add first-aha happy path and error state tests`
11. `test(web): add unhealthy sync apply-disabled tests`
12. `docs(ux): document first aha flow`
