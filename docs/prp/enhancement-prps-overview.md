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
   - Uses existing Basic Auth/env credentials for MVP.
   - Required before asking users to trust plan writes.

2. **Planning Safety Skeleton, Diff, Audit Log, and Rollback**
   - Builds the backend trust loop.
   - Requires no real LLM.
   - Gives frontend stable APIs.
   - Defines the apply response contract used by First Aha UX.

3. **First Aha UX - Agenda, Focus Proposal, Diff Review, and Undo**
   - Turns the backend skeleton into the first visible product moment.
   - Can start with mocked APIs, but should ship after proposal/apply/rollback APIs stabilize.
   - Do not begin First Aha implementation until PRP 2 API response contracts are stable.

## Dependency map

```text
CalDAV trust
├── Sync Health API
├── Debug bundle
├── Green Sync onboarding
├── Basic Auth/env credential guidance
└── Interop evidence

Planning safety
├── Depends on existing EventRepo/CalDAV storage
├── Uses ETags from CalDAV foundation
├── Adds PlanProposal/Diff/Apply/Rollback
├── Adds canonical stale proposal state
├── Writes audit log
└── Defines apply response contract

First Aha UX
├── Depends on Sync Health status and green-sync completion
├── Depends on proposal/apply/rollback/audit APIs
├── Depends on PRP 2 apply response contract:
│   ├── audit_log_entry_id
│   ├── rollback_available
│   ├── can_apply
│   └── applied change summary
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

## Interop cutline

- Apple Calendar, Fantastical, and DAVx5 must all be represented in `docs/testing/caldav-interop-results.md`.
- At least two of the three must pass manual CRUD validation for MVP.
- Non-passing or not-yet-tested clients must be documented with status, reason, and follow-up issue.
- Thunderbird remains optional for MVP and should be documented if easy.

## Definition of Ready for Dev Execution

Before implementation begins for each PRP:

- [ ] Goals and non-goals are explicit.
- [ ] MVP vs post-MVP scope is clear.
- [ ] API contracts are defined.
- [ ] Data model changes are identified or explicitly not required.
- [ ] Acceptance criteria are objective and testable.
- [ ] Manual validation steps are documented.
- [ ] Cross-PRP dependencies are satisfied.
- [ ] Privacy/redaction requirements are specified.
- [ ] No hidden LLM or Todoist dependency exists.

Do not begin First Aha implementation until PRP 2 API response contracts are stable.

## Suggested epic breakdown

### Epic 1 - CalDAV trust and diagnostics

1. `feat(server): add CalDAV operation metadata logging`
2. `feat(server): add sync health summary service`
3. `feat(api): expose sync health endpoints`
4. `feat(server): add redacted authenticated debug bundle export`
5. `feat(web): add Sync Health dashboard`
6. `feat(web): add CalDAV onboarding gate using Basic Auth/env credentials`
7. `test(caldav): add validation-event flow tests`
8. `test(sync): add health status and retention tests`
9. `docs(testing): publish CalDAV interop matrix`

### Epic 2 - Planning safety skeleton

1. `feat(planner): add proposal/diff/validation domain types`
2. `feat(planner): implement canonical proposal statuses including stale`
3. `feat(planner): implement deterministic constraint engine`
4. `feat(planner): implement supported change type validation`
5. `feat(planner): implement focus slot finder`
6. `feat(data): persist proposals and rollback payloads`
7. `feat(api): add daily proposal endpoint`
8. `feat(api): add proposal apply endpoint with revalidation and response contract`
9. `feat(api): add rollback endpoint`
10. `feat(api): add audit log endpoints with redacted responses`
11. `test(planner): cover constraints, stale proposals, unsupported changes, and rollback conflicts`

### Epic 3 - First Aha UX

1. `feat(api): add today endpoint`
2. `feat(web): add Today dashboard`
3. `feat(web): add agenda timeline`
4. `feat(web): add click-to-generate proposal card`
5. `feat(web): add diff preview`
6. `feat(web): add apply flow with sync-health gate`
7. `feat(web): add undo flow using rollback_available and audit_log_entry_id`
8. `feat(web): add reject proposal dialog`
9. `test(web): add first-aha UX tests`
10. `test(web): add unhealthy sync apply-disabled tests`
11. `docs(ux): document first aha flow`

## MVP cutline

### Must ship

- CalDAV green-sync onboarding gate.
- MVP onboarding using existing Basic Auth/env credentials.
- Sync Health dashboard with deterministic Healthy / Warning / Critical / Unknown rules.
- Authenticated redacted debug bundle.
- Interop doc representing Apple Calendar, Fantastical, and DAVx5, with at least two passing manual CRUD.
- Deterministic proposal model.
- Canonical `stale` proposal status.
- Constraint validation before display and before apply.
- Machine-readable diff.
- Human-readable diff.
- Apply endpoint with ETag revalidation.
- Apply response with `audit_log_entry_id`, `rollback_available`, `can_apply`, and applied change summary.
- Audit log entry for every apply.
- Rollback endpoint with external-change conflict blocking.
- Audit APIs that do not expose raw ICS by default.
- Today agenda.
- One click-generated focus block recommendation.
- Apply disabled unless `can_apply`, validation, sync health, and green-sync gates pass.
- Undo UI shown only when `rollback_available === true`.
- View audit log link using `audit_log_entry_id`.
- Rejection reason capture.
- No raw ICS in default logs/debug bundles.

### Can defer

- App-specific CalDAV passwords and per-client credentials.
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
| Debug bundle exposed without auth | CalDAV trust | Critical | Require authenticated local admin/session access |
| Rollback unsafe after external edits | Planning safety, First Aha UX | High | ETag-at-apply checks before rollback |
| UI implies AI autonomy | First Aha UX | High | Agenda-first, explicit Apply, clear diff, explicit click-to-generate |
| Sync unhealthy still allows writes | First Aha UX | High | Hard-disable Apply unless sync and green-sync gates pass |
| CalDAV client quirks delay UX | CalDAV trust | Medium | 2-of-3 pass gate, document all three, defer full RFC 6578 |
| Proposal feels underwhelming | First Aha UX | Medium | Make trust the magic: validation, diff, undo, visible calendar write |
| Backend schema churn | Planning safety | Medium | Store JSON payloads for proposal/rollback while stabilizing |
| Scope creep into full planner | Planning safety, First Aha UX | High | One focus block only, no Todoist writes, no real LLM |
