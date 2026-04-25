# PRP: First Aha UX - Agenda, Focus Proposal, Diff Review, and Undo

**Type:** Implementation PRP / Product Requirements Prompt  
**Target Team:** Frontend + Backend API + UX QA  
**Status:** Proposed  
**Priority:** MVP user-visible trust loop

---

## 1. Context

The first visible Calendar-app aha should be small, safe, and concrete: Calendar-app finds one open slot, proposes one protected focus block, validates it, shows the diff, applies after confirmation, logs it, and can undo it.

This PRP turns the backend planning safety skeleton into an agenda-first user experience. It depends on:

- Sync Health status from the CalDAV trust PRP.
- Proposal/diff/apply/audit/rollback APIs from the planning safety PRP.

The experience should work without a real LLM and without Todoist. The point is to make the trust promise visible.

---

## 2. Problem statement

Users need to experience Calendar-app's trust promise in the UI, not just in backend safety code. A chat-first or full-day planning experience is too broad for the first demo. The MVP should show one validated focus block proposal in an agenda-first dashboard, with visible reasoning, diff, apply, audit, and undo.

---

## 3. Goals

- Create Today dashboard with agenda and sync health status.
- Show one recommended focus block proposal.
- Explain why the slot was chosen.
- Show constraints checked and validation status.
- Show a clear diff before apply.
- Allow apply only when proposal is valid.
- Show success banner with Undo and View audit log.
- Capture rejection reason.
- Handle no-slot, unhealthy sync, invalid proposal, apply failure, and rollback failure states.

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

---

## 5. User stories

1. As a user, I can open Today and see my agenda plus sync health.
2. As a user, I see one suggested protected focus block that fits my day.
3. As a user, I can understand why the slot was chosen and what constraints passed before applying.
4. As a user, I can open a diff and explain what will change.
5. As a user, I can apply the proposal only after confirmation.
6. As a user, I can undo the applied focus block if nothing changed externally.
7. As a user, I can reject a bad proposal with a structured reason.
8. As a user, I understand what to do when sync is unhealthy, no slot exists, proposal is invalid, apply fails, or rollback is blocked.

---

## 6. Functional requirements

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
   - Optional LLM state if visible: not configured / configured.
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
   - One proposal.
   - Explanation.
   - Validation.
   - Diff summary.
   - Actions.

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

Proposal card must show:

```text
Suggested protection:
Add "Deep Work" from 9:30-11:30

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
- Do not show Apply if `validation_result.valid=false`.
- If proposal expires, show `Regenerate proposal`.
- If sync status is unhealthy, show warning before proposal generation: `Calendar sync is unhealthy. Fix sync before applying plans.`
- If no LLM key exists, do not block first proposal; show `Using deterministic planner.`

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
- Expires at.

### FR7 - Apply confirmation

Apply behavior:

- Button label: `Apply focus block`.
- On click, show lightweight confirmation if only adding a focus block.
- Strong confirmation is reserved for destructive changes.
- Loading state disables buttons and shows `Revalidating and applying...`.

Success:

- Add focus block to agenda.
- Show banner:

```text
Focus block added.

[Undo] [View audit log]
```

Failure:

- Show specific error:
  - Stale proposal.
  - Validation failed.
  - ETag changed.
  - Server error.
  - Sync unhealthy.

### FR8 - Undo after apply

Undo action:

- Calls `/api/v1/plan/rollback`.

On success:

- Remove focus block from agenda.
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

- Provide link to audit log entry.

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
4. Proposal invalid
   - Show violations; no Apply.
5. Apply failed
   - Show error and next step.
6. Rollback failed
   - Show reason and link to audit log.
7. LLM not configured
   - `Using deterministic planner. LLM setup is optional.`
8. Proposal expired
   - `This proposal expired because your calendar may have changed.`
   - Show regenerate action.

---

## 7. Non-functional requirements

- Trust clarity: user must be able to identify what will change before applying.
- Accessibility: keyboard-accessible modal/drawer, focus trap, ARIA labels, WCAG 2.1 AA contrast.
- Responsiveness: desktop-first, responsive enough for mobile review/apply.
- Performance: Today dashboard loads under 1s locally for typical single-user data.
- No silent writes: every apply originates from explicit user action.
- Resilience: UI handles API errors without losing local state.
- Privacy: do not expose raw ICS in technical details.
- State correctness: agenda updates after apply/undo via refetch or local mutation confirmed by server response.

---

## 8. Technical approach

### Frontend architecture

Components:

```text
TodayPage
  TodayHeader
  SyncStatusBadge
  AgendaTimeline
  AvailabilitySummary
  FocusProtectionSummary
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

### Backend additions

Add `GET /api/v1/today` if missing:

- Uses EventRepo to list events by date.
- Uses SyncHealthService for status summary.
- Computes availability from working hours and events.
- Computes focus summary.

---

## 9. Files likely to change

Frontend:

```text
web/src/pages/TodayPage.tsx
web/src/components/today/TodayHeader.tsx
web/src/components/today/AgendaTimeline.tsx
web/src/components/today/AvailabilitySummary.tsx
web/src/components/today/FocusProtectionSummary.tsx
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

## 10. API changes

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

---

## 11. Database/schema changes

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

## 12. UX changes

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

- Suggested protection card.

Below or side:

- Availability summary.
- Focus protection summary.

### Proposal card copy

Use plain language:

- `Suggested protection`
- `Why this slot`
- `Validation passed`
- `What will change`
- `Apply focus block`
- `Reject`

Avoid:

- `AI optimized your schedule`
- `Autopilot`
- `We changed your calendar`

### Accessibility requirements

- All buttons keyboard reachable.
- Diff drawer closes with Escape.
- Focus returns to opener after drawer closes.
- Validation status announced to screen readers.
- Loading states use `aria-busy`.
- Error banners use `role="alert"`.

---

## 13. Test plan

Frontend unit/component tests:

- Today dashboard renders agenda.
- Sync status badge renders healthy/warning/critical.
- Proposal card renders explanation and validation.
- Apply hidden/disabled for invalid proposal.
- Diff preview opens and groups changes.
- Apply success shows Undo banner.
- Undo success removes banner/refetches agenda.
- Rollback blocked shows clear error.
- Reject dialog submits reason.
- No open slot empty state renders.
- Sync unhealthy warning links to Sync Health.

Frontend integration tests:

- Mock full happy path: load today, generate proposal, open diff, apply, see success, undo.
- Mock stale proposal apply failure.
- Mock invalid proposal.

Backend tests:

- `GET /api/v1/today` returns events and availability.
- Availability excludes fixed events.
- Availability computes longest open slot.
- Today endpoint includes sync status.

UX test:

- Give user a proposal screen.
- Ask: `What will change if you click Apply?`
- Pass if user says it will add a Deep Work focus block from X to Y and does not think existing events will move/delete.

---

## 14. Manual validation steps

1. Complete green sync from the CalDAV trust PRP.
2. Seed today calendar:
   - Standup 08:30-09:00.
   - Lunch 12:00-13:00.
   - Meeting 14:00-15:00.
3. Open `/today`.
4. Confirm agenda appears.
5. Confirm Sync Health status appears.
6. Generate/view proposal.
7. Confirm proposal suggests one safe focus block.
8. Open diff.
9. Confirm diff says only one event will be added.
10. Apply.
11. Confirm focus block appears in agenda and external CalDAV client.
12. Confirm audit log entry exists.
13. Click Undo.
14. Confirm focus block disappears from agenda and external CalDAV client.
15. Repeat with intentionally invalid proposal fixture; confirm no Apply.
16. Repeat with external edit after apply; confirm Undo blocked clearly.

---

## 15. Acceptance criteria

- User can view Today agenda.
- User can see Sync Health status.
- User can generate or view one proposed focus block.
- Proposal includes explanation and validation status.
- Proposal shows constraints checked.
- User can open diff preview.
- User can apply the proposal.
- Applied focus block appears on the calendar.
- Audit log entry is created.
- Success banner includes Undo and View audit log.
- User can undo the apply if no external conflict exists.
- Undo blocked state is clear when external ETag changed.
- Invalid proposal does not show actionable Apply.
- Reject flow captures structured reason and optional free text.
- No LLM key is required for first aha.
- UX test passes: user can explain what will change before applying.
- `go test ./...` passes.
- Frontend build/test passes using project-standard commands.

---

## 16. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| First aha feels too small | Medium | Make trust visible: why, validation, diff, undo |
| UI overexplains | Medium | Default concise card; technical details expandable |
| Apply race causes failure | Medium | Treat stale proposal as expected state with regenerate CTA |
| Undo blocked surprises user | Medium | Explain external changes and link to audit log |
| Agenda UI becomes scope creep | High | No drag/drop; no week view; Today only |
| No-slot state feels like failure | Low | Explain blockers and suggest retry/edit preferences later |

---

## 17. Open questions

- Should the first focus block default to 90 minutes or 2 hours?
- Should `Deep Work` title be configurable before apply or fixed for MVP?
- Should focus blocks use a dedicated calendar if one exists?
- Should the UI auto-generate a proposal on page load or require `Find focus slot` click?
- Should success banner persist until dismissed or disappear after navigation?

---

## 18. Suggested GitHub issue breakdown

1. `feat(api): add today dashboard endpoint`
2. `feat(web): add Today dashboard route`
3. `feat(web): add agenda timeline component`
4. `feat(web): add sync status and availability summaries`
5. `feat(web): add proposal card with explanation and validation`
6. `feat(web): add diff preview drawer`
7. `feat(web): add apply proposal flow`
8. `feat(web): add undo rollback banner`
9. `feat(web): add rejection reason dialog`
10. `test(web): add first-aha happy path and error state tests`
11. `docs(ux): document first aha flow`
