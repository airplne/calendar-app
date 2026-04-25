# Calendar-app Enhancement PRPs Overview

This document indexes the three next implementation PRPs for sharpening Calendar-app around the MVP wedge:

> A self-hosted calendar source of truth that protects focus time with explainable, reversible AI proposals.

## PRPs

1. [CalDAV Trust, Sync Health, and Interop Gate](./caldav-trust-sync-health-interop-gate.md)
2. [Planning Safety Skeleton, Diff, Audit Log, and Rollback](./planning-safety-diff-audit-rollback.md)
3. [First Aha UX - Agenda, Focus Proposal, Diff Review, and Undo](./first-aha-agenda-focus-diff-undo.md)

## Recommended implementation order

1. **CalDAV Trust, Sync Health, and Interop Gate**
   - Proves the calendar source of truth.
   - Adds diagnostics and onboarding gate.
   - Required before asking users to trust plan writes.

2. **Planning Safety Skeleton, Diff, Audit Log, and Rollback**
   - Builds the backend trust loop.
   - Requires no real LLM.
   - Gives frontend stable APIs.

3. **First Aha UX - Agenda, Focus Proposal, Diff Review, and Undo**
   - Turns the backend skeleton into the first visible product moment.
   - Can start with mocked APIs, but should ship after proposal/apply/rollback APIs stabilize.

## Dependency map

```text
CalDAV trust
├── Sync Health API
├── Debug bundle
├── Green Sync onboarding
└── Interop evidence

Planning safety
├── Depends on existing EventRepo/CalDAV storage
├── Uses ETags from CalDAV foundation
├── Adds PlanProposal/Diff/Apply/Rollback
└── Writes audit log

First Aha UX
├── Depends on Sync Health status
├── Depends on proposal/apply/rollback/audit APIs
├── Adds Today dashboard
└── Delivers first user-visible aha
```

Critical path:

```text
Raw ICS + ETag correctness
-> Sync Health / green sync
-> Deterministic proposal + validation
-> Apply + audit + rollback
-> Today UI + diff + undo
```

## Suggested epic breakdown

### Epic 1 - CalDAV trust and diagnostics

1. `feat(server): add CalDAV operation metadata logging`
2. `feat(server): add sync health summary service`
3. `feat(api): expose sync health endpoints`
4. `feat(server): add redacted debug bundle export`
5. `feat(web): add Sync Health dashboard`
6. `feat(web): add CalDAV onboarding gate`
7. `test(caldav): add validation-event flow tests`
8. `docs(testing): publish CalDAV interop matrix`

### Epic 2 - Planning safety skeleton

1. `feat(planner): add proposal/diff/validation domain types`
2. `feat(planner): implement deterministic constraint engine`
3. `feat(planner): implement focus slot finder`
4. `feat(data): persist proposals and rollback payloads`
5. `feat(api): add daily proposal endpoint`
6. `feat(api): add proposal apply endpoint with revalidation`
7. `feat(api): add rollback endpoint`
8. `feat(api): add audit log endpoints`
9. `test(planner): cover constraints, stale proposals, rollback conflicts`

### Epic 3 - First Aha UX

1. `feat(api): add today endpoint`
2. `feat(web): add Today dashboard`
3. `feat(web): add agenda timeline`
4. `feat(web): add proposal card`
5. `feat(web): add diff preview`
6. `feat(web): add apply and undo flows`
7. `feat/web): add reject proposal dialog`
8. `test(web): add first-aha UX tests`
9. `docs(ux): document first aha flow`

## MVP cutline

### Must ship

- CalDAV green-sync onboarding gate.
- Sync Health dashboard.
- Redacted debug bundle.
- Interop doc with Apple/Fantastical/DAVx5 results.
- Deterministic proposal model.
- Constraint validation before display and before apply.
- Machine-readable diff.
- Human-readable diff.
- Apply endpoint with ETag revalidation.
- Audit log entry for every apply.
- Rollback endpoint with external-change conflict blocking.
- Today agenda.
- One focus block recommendation.
- Apply and Undo UI.
- Rejection reason capture.
- No raw ICS in default logs/debug bundles.

### Can defer

- Thunderbird interop if Apple/Fantastical/DAVx5 coverage is strong.
- Full RFC 6578 `sync-collection`.
- Real LLM planner.
- Todoist writes.
- Todoist read sync in first aha.
- Multi-option proposal ranking.
- Drag/drop calendar editing.
- Week view.
- Chat interface.
- Destructive move/delete proposal UI.
- Slack focus status.
- Hosted multi-tenant deployment.
- Advanced recurrence modifications.
- Preference learning beyond rejection capture.

## Cross-PRP risks

| Risk | Affected PRP | Severity | Mitigation |
|---|---|---:|---|
| Calendar data loss | All | Critical | Raw ICS source of truth, ETags, transactions, interop tests |
| Privacy leak in logs/debug/audit | CalDAV trust, planning safety | Critical | Redaction by default, tests, raw export opt-in |
| Rollback unsafe after external edits | Planning safety, First Aha UX | High | ETag-at-apply checks before rollback |
| UI implies AI autonomy | First Aha UX | High | Agenda-first, explicit Apply, clear diff |
| CalDAV client quirks delay UX | CalDAV trust | Medium | 2-of-3 pass gate, document quirks, defer full RFC 6578 |
| Proposal feels underwhelming | First Aha UX | Medium | Make trust the magic: validation, diff, undo, visible calendar write |
| Backend schema churn | Planning safety | Medium | Store JSON payloads for proposal/rollback while stabilizing |
| Scope creep into full planner | Planning safety, First Aha UX | High | One focus block only, no Todoist writes, no real LLM |
