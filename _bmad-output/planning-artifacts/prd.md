---
stepsCompleted:
  - step-01-init
  - step-02-discovery
  - step-03-success
inputDocuments: []
workflowType: 'prd'
documentCounts:
  briefs: 0
  research: 0
  brainstorming: 0
  projectDocs: 0
classification:
  projectType: API Backend + Web App
  domain: General/Productivity
  complexity: Medium (MVP-scoped)
  projectContext: Greenfield
  platformStrategy: Linux-first, CalDAV clients for mobile
  notes:
    - Fantastical is a CalDAV client (not direct API integration)
    - Mobile access via existing CalDAV clients + responsive web UI
    - CalDAV scoped to minimal interoperable subset for MVP
    - Full CalDAV spec (recurrence, sync tokens, conflict resolution) is expansion
---

# Product Requirements Document - Calendar-app

**Author:** Daniel
**Date:** 2026-01-15

## Success Criteria

### User Success

**The "Aha!" Moment:**
- First time AI proposes a plan that actually works - tasks fit realistically into available time
- When a new meeting lands, the system automatically detects it and proposes an updated plan immediately (1-click apply)
- Both together prove the value proposition

**Relief Scenario (ADHD/Context-Switching Users):**
- Cognitive offload - stop holding "what should I do next?" in their head
- Protected focus - finish deep work blocks without interruption anxiety
- The system both thinks for them AND guards their execution time

**End-of-Day/Week "Done" State:**
- Completion confidence - know what got done, what moved, and why
- No guilt about unfinished tasks because tradeoffs were explicit
- Sustainable pace - fewer surprise crunches, energy left over

### Business Success (People & Outcomes)

**3-Month Indicators:**
- Retention: Users who try it keep using it daily; AI becomes indispensable
- Organic growth: Word-of-mouth brings new users without marketing spend

**12-Month Indicators:**
- Life impact: Users report less burnout, better work-life boundaries, reclaimed time
- Behavior change: Shift from reactive scheduling to proactive planning; users think differently about time

### Technical Success

- **Reliability:** CalDAV sync never loses data; hard-constraint validation is deterministic and runs before display/apply - no plan that violates calendar hard constraints (overlaps with fixed events, outside allowed windows, etc.) is ever shown as actionable or allowed to apply; complete user trust
- **Responsiveness:** Plan generation and adaptation feel instant; no waiting for AI
- **CalDAV Interoperability:** CRUD + sync works with Fantastical, Apple Calendar, and at least one Android CalDAV client (no data loss)

### Measurable Outcomes

| Metric | Target | Definition |
|--------|--------|------------|
| Daily active retention (30-day) | >60% of activated users | **Activated user:** completed onboarding + connected Todoist + received at least one AI-generated plan |
| Plan acceptance rate | >80% accepted or minimally adjusted | AI proposals accepted as-is or with ≤2 manual edits |
| Focus block completion | >70% uninterrupted | **Completion:** user marks focus block done OR no calendar conflicts created during the block's scheduled time |
| Sync reliability | Zero data loss incidents | No events/tasks lost or corrupted across CalDAV sync cycle |
| Plan generation latency | <2 seconds | Time from trigger (new day, meeting change) to plan displayed |
| CalDAV interop | 3+ clients verified | Fantastical + Apple Calendar + 1 Android client pass CRUD + sync tests |

## Product Scope

### MVP - Minimum Viable Product

**Core Loop:**
- CalDAV server (minimal interoperable subset)
- Todoist read/write sync
- AI proposes daily plan
- User confirms/adjusts
- Changes apply to calendar + tasks

**Protection System:**
- Focus time guarding
- Overload detection
- Automatic adaptation when new meetings land (propose updated plan, 1-click apply)

### Growth Features (Post-MVP)

- Multi-calendar intelligence (work + personal unified)
- Life context awareness beyond just work
- Integration expansion beyond Todoist (other task managers, note apps, email)
- Become the central planning hub

### Vision (Future)

- Team orchestration (shared calendars, cross-team availability, AI coordinates multiple people)
- Predictive intelligence (learns patterns, preemptive blocking, burnout warnings)

