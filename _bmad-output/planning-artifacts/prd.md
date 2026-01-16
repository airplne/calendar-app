---
stepsCompleted:
  - step-01-init
  - step-02-discovery
  - step-03-success
  - step-04-journeys
  - step-05-domain
  - step-06-innovation
  - step-07-project-type
  - step-08-scoping
  - step-09-functional
  - step-10-nonfunctional
  - step-11-polish
  - step-12-complete
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

## User Journeys

### Journey 1: Marcus - The Meeting-Heavy Executive

**Persona:**
- **Name:** Marcus, 42, VP of Product at a growth-stage startup
- **Situation:** 6-8 meetings daily, Slack fires constantly, strategic work happens at 10pm or not at all
- **Goal:** Ship the Q2 roadmap without burning out or dropping balls
- **Obstacle:** No time to think; every "free" 30 minutes gets sniped by someone's "quick sync"

**Opening Scene:**
Marcus stares at Monday's calendar - wall-to-wall from 9am to 6pm. His Todoist has 47 tasks, 12 overdue. He used to block "focus time" but it never survived first contact with his team. He's been meaning to write the Q2 strategy doc for two weeks.

**Rising Action:**
Marcus sets up Calendar-app's CalDAV server (source of truth) and points Fantastical/Apple Calendar/Android CalDAV client(s) at it, then connects Todoist. The AI analyzes his week: "You have 2.3 hours of genuinely available time across 5 days. I found 3 tasks that could fit. The strategy doc needs ~4 hours - should I protect Thursday afternoon by declining or proposing reschedules for 2 lower-priority meetings?"

Marcus hesitates, then clicks "Propose reschedules." Calendar-app proposes options and drafts reschedule messages for each meeting. For the one where Marcus is organizer, he confirms and the change applies immediately. For the other, Calendar-app drafts the request for Marcus to send; once the organizer updates the invite, the change syncs via CalDAV and his calendar updates. Thursday 2-6pm is now protected.

**Climax:**
Wednesday at 3pm, his CEO adds an "urgent board prep" meeting for Thursday at 3pm - right in his protected block. Within 30 seconds, Calendar-app notifies him: "Your focus block was interrupted. Options:

**(C)** Split it - 2-3pm Thursday + Friday 10am-12pm
**(B)** Protect it - draft message suggesting CEO alternative times
**(CB)** Partial split + partial protection - keep 2-3pm, draft proposal to move CEO to 4pm
**(BC)** Sacrifice it - reschedule strategy doc to next week"

He picks C. The AI proposes the rebalanced plan; Marcus confirms with 1-click and the changes apply.

**Resolution:**
Thursday evening, Marcus has 60% of the strategy doc done. Not perfect, but progress. He didn't work at 10pm. He tags his CEO as "immovable" in Calendar-app's settings; the system remembers this preference and won't propose rescheduling CEO meetings in future plans.

---

### Journey 2: Priya - The Deep-Work Creator

**Persona:**
- **Name:** Priya, 34, senior software engineer at a remote-first company
- **Situation:** Her best work happens in 3-4 hour uninterrupted blocks; she's building a complex distributed system
- **Goal:** Ship the new caching layer without context-switching destroying her flow state
- **Obstacle:** Standups, 1:1s, and "quick questions" fragment her day into useless 45-minute chunks

**Opening Scene:**
Priya looks at Tuesday - three meetings scattered across the day at 10am, 1pm, and 4pm. Each is only 30 minutes, but they shatter any hope of deep work. Her Todoist shows "Implement cache invalidation" has been carried forward for 8 days. She knows it needs 3 focused hours minimum.

**Rising Action:**
She sets up Calendar-app's CalDAV server and syncs it with Apple Calendar on her Mac, then connects Todoist. She configures her preferences: "Minimum focus block: 2 hours. Preferred deep work: mornings. Meetings: batch if possible."

The AI analyzes: "Your Tuesday has 3 meetings creating 4 fragmented slots. I can propose batching your 1pm and 4pm into back-to-back at 1pm - this opens a 3-hour focus block from 2:30-5:30pm. Want me to draft a reschedule request for the 4pm?"

Priya confirms. Calendar-app drafts the reschedule request for Priya to review/send (or copy/paste); since she's not the organizer, she sends it to the meeting owner. The owner accepts and updates the invite, which syncs via CalDAV, and Priya's afternoon opens up.

**Climax:**
Later that week, a product manager drops a "quick 15-min sync" at 10:30am - right in her morning focus block. Calendar-app flags it immediately:

"Your 9am-12pm focus block was interrupted. Options:

**(C)** Defend - draft message suggesting PM alternative times outside your focus window
**(B)** Absorb - accept the 15 min, split focus into 9-10:15am + 10:45am-12pm
**(CB)** Negotiate - draft async proposal (Loom/Slack) instead of sync meeting
**(BC)** Accept - allow the interruption as-is"

She picks CB. Calendar-app drafts: "Hey! I'm in deep focus 9-12. Could we do this async? Happy to Loom my answer or respond in Slack by noon." Priya reviews and sends.

**Resolution:**
The PM agrees to async. Priya finishes cache invalidation by 11:30am. She marks "morning focus blocks" as high-protection; Calendar-app remembers her preference for future planning.

---

### Journey 3: Sync Conflict - Marcus's Double-Edit

**Scenario:**
Marcus is reviewing his week in Calendar-app's web UI while simultaneously his assistant edits the same shared calendar/event from another CalDAV client (Fantastical) and moves the meeting to 4pm.

**Opening Scene:**
Marcus sees a 2pm meeting on Thursday and drags it to 3pm in Calendar-app. At the same moment, his assistant edits the same shared calendar/event from another CalDAV client (Fantastical) and moves the meeting to 4pm.

**Rising Action:**
Both changes hit the CalDAV server within seconds of each other. Calendar-app detects the conflict during sync reconciliation.

**Climax:**
Calendar-app surfaces the conflict to Marcus:

"Sync conflict detected for 'Product Review' on Thursday:

**(C)** Keep your change (3pm) - overwrite the 4pm version
**(B)** Keep external change (4pm) - discard your 3pm move
**(CB)** Review both - show me a diff before deciding
**(BC)** Other - I need to check with my assistant first"

Marcus picks CB. Calendar-app shows: "You moved it to 3pm at 10:42am. External change to 4pm at 10:43am from Fantastical." He realizes his assistant had a reason and picks B.

**Resolution:**
The 4pm version wins. Calendar-app logs the resolution for his reference. No data is lost - both versions were preserved until Marcus decided. The AI re-proposes his daily plan accounting for the 4pm meeting.

---

### Journey 4: AI Rejection Loop - Priya's "This Doesn't Work"

**Scenario:**
Priya repeatedly rejects AI-proposed plans because they don't account for her unlisted constraints.

**Opening Scene:**
Calendar-app proposes Priya's Wednesday plan: "Focus block 9am-12pm for cache implementation, then meetings 1-3pm, then 'Write documentation' 3:30-5pm."

Priya frowns. She rejects it: "No - I can't do documentation after meetings. My brain is fried by 3pm."

**Rising Action:**
Calendar-app proposes alternative: "Focus block 9-11am for cache, documentation 11am-12pm, meetings 1-3pm, then light tasks 3:30-5pm."

Priya rejects again: "Documentation needs at least 90 minutes uninterrupted, not 60."

Calendar-app tries again: "Documentation 9-10:30am, cache work 10:30am-12pm, meetings 1-3pm, light tasks after."

Priya rejects: "No - cache work is my highest priority this week. It has to come first in the morning when I'm sharpest."

**Climax:**
Calendar-app recognizes the pattern and asks:

"I'm having trouble finding a plan that works. Help me understand your constraints:

**(C)** Capture preferences - let me tell you my rules (energy levels, task priorities, time-of-day preferences)
**(B)** Manual override - I'll build today's plan myself, you watch and learn
**(CB)** Guided setup - walk me through a few questions to capture my work patterns
**(BC)** Skip for now - just give me the least-bad option and I'll adjust"

Priya picks CB. Calendar-app asks: "When is your peak mental energy?" (Morning), "What's your minimum focus block?" (2 hours), "After meetings, are you better with creative work or administrative tasks?" (Administrative/light tasks only).

**Resolution:**
Calendar-app stores Priya's explicit preferences: peak energy = morning, minimum focus = 2 hours, post-meeting = light tasks only. The next proposal respects all three rules. Priya accepts with one small tweak. Future plans consistently work on the first or second try.

---

### Journey 5: Self-Hoster/Operator - DevOps Dana Sets Up Calendar-app

**Persona:**
- **Name:** Dana, 38, freelance DevOps consultant
- **Situation:** Runs her own Linux server for personal infrastructure (Nextcloud, email, etc.); wants calendar sovereignty
- **Goal:** Get Calendar-app running on her VPS, connect all her devices, and stop depending on Google Calendar
- **Obstacle:** CalDAV servers are notoriously finicky; she's been burned by sync bugs before

**Opening Scene:**
Dana has a Debian VPS with Docker already running. She's tired of Google owning her schedule data. She finds Calendar-app, reads the README, and sees "Linux-first, self-hosted CalDAV server with AI planning."

**Rising Action:**
Dana deploys Calendar-app via Docker Compose (or a binary) and configures it via an env/config file: storage path for calendar data, domain/port for the CalDAV endpoint. Todoist API token and LLM provider key are optional - she skips them for now to verify CalDAV works first.

She configures nginx as a reverse proxy with SSL. Calendar-app provides the CalDAV URL format: `https://calendar.dana.dev/caldav/`

Dana opens Fantastical on her Mac, adds a CalDAV account with her credentials. Events sync. She adds the same account to Apple Calendar on her iPhone and DAVx⁵ on her Android tablet.

**Climax:**
First sync issue: DAVx⁵ on Android shows duplicate events. Dana checks Calendar-app's logs:

"Sync diagnostic available:

**(C)** View sync log - show recent CalDAV operations with client fingerprints
**(B)** Check client compatibility - known issues with detected clients
**(CB)** Export debug bundle - downloadable log package for troubleshooting
**(BC)** Skip - I'll figure it out myself"

She picks C. Logs show DAVx⁵ is triggering duplicate writes/requests (a known client quirk). Calendar-app flags it and links to known mitigation/settings guidance for DAVx⁵.

**Resolution:**
Dana applies the recommended settings fix; duplicates stop. All three clients sync cleanly. She adds the Todoist token and LLM API key to the config now that CalDAV is stable. She sets up a cron job for daily backups of the calendar data directory. Calendar-app is now her source of truth - no vendor lock-in, full data ownership.

---

### Journey 6: API/Integration Developer - Tomas Builds a Slack Bot

**Persona:**
- **Name:** Tomas, 29, full-stack developer at a small agency
- **Situation:** Uses Calendar-app personally; wants to build a Slack bot that shows focus time availability
- **Goal:** Query Calendar-app to check if he (or opted-in teammates) are in a focus block before being interrupted
- **Obstacle:** Needs programmatic access without breaking CalDAV sync or the AI planning layer

**Opening Scene:**
Tomas's team has a culture problem: people interrupt each other constantly via Slack. He wants a `/busy` command that checks Calendar-app and responds with focus status. For MVP, he starts with self-check (`/busy` returns your own status). Post-MVP, he envisions `/busy @teammate` for users who explicitly opt-in and grant the bot permission.

**Rising Action:**
Tomas checks Calendar-app's documentation. He finds:
- CalDAV subset for interoperability is the primary sync protocol
- Calendar-app exposes an integration API (read-only) for querying derived state
- Focus blocks are represented as calendar events (tagged or on a dedicated calendar); the integration API derives "focus status" from CalDAV state - no separate authoritative store
- Write operations go through CalDAV or the web UI to maintain single source of truth

He registers an API key scoped to read-only access for his user. An illustrative endpoint like `GET /api/v1/me/focus-status` might return:
```json
{
  "in_focus_block": true,
  "block_name": "Deep Work - Cache Implementation",
  "ends_at": "2026-01-15T14:00:00Z",
  "interruptible": false
}
```

**Climax:**
Tomas builds the Slack bot (MVP: self-check only). First test: he runs `/busy` while in a focus block.

The bot responds: "You are in 'Deep Work - Cache Implementation' until 2:00pm (1h 23m remaining). Marked as non-interruptible.

**(C)** Share status - post to channel so teammates know
**(B)** Set reminder - ping me when focus block ends
**(CB)** Snooze notifications - mute Slack until block ends
**(BC)** Dismiss - just checking"

He picks C. The channel sees: "Tomas is in deep focus until 2pm."

**Resolution:**
The team adopts the self-check bot. Tomas plans v2: teammates can opt-in via Calendar-app settings ("Allow focus status queries from: [bot name]"), enabling `/busy @tomas`. He notes that the integration API is read-only by design - any scheduling changes go through the user's Calendar-app UI or CalDAV client, and focus status is always derived from the calendar source of truth.

---

### Journey Requirements Summary

| Capability Area | Revealed By Journey | Requirements |
|-----------------|---------------------|--------------|
| **CalDAV Server** | Dana (5), All | Minimal interoperable CalDAV subset for MVP (CRUD + basic sync with major clients); full recurrence/sync-token completeness/conflict edge coverage is expansion scope unless explicitly included |
| **Todoist Integration** | Marcus (1), Priya (2) | Read tasks/projects; write updates back to Todoist (priority, due dates, labels/sections/comments as needed) and link tasks to planned calendar time blocks (calendar remains the time-block source of truth) |
| **AI Planning Engine** | Marcus (1), Priya (2), Rejection (4) | Propose daily plans; respect hard constraints; adapt when calendar changes; <2s latency |
| **Focus Block Protection** | Marcus (1), Priya (2), Tomas (6) | Detect interruptions; propose options (split/defend/absorb); track completion |
| **Preference System** | Priya (2), Rejection (4) | Store user rules (energy patterns, minimum blocks, post-meeting preferences, immovable tags) |
| **Propose → Confirm → Apply Loop** | Marcus (1), Priya (2) | Draft reschedule messages; apply changes only after user confirmation; respect organizer permissions |
| **Conflict Resolution UI** | Sync Conflict (3) | Surface conflicts; show diff; let user decide winner; log resolution |
| **Operator Tooling** | Dana (5) | Env/config file deployment; sync logs with client fingerprints; debug bundle export; backup-friendly data directory |
| **Internal REST API** | Marcus (1), Priya (2), All | Used by Web UI for planning operations, preferences, Todoist sync; not marketed as third-party API |
| **Public Integration API** | Tomas (6) | Minimal read-only (MVP-lite): self-check focus status; full integration platform (team queries, expanded endpoints, SDKs) deferred to Post-MVP |

## Domain-Specific Requirements

### Data Privacy

**Local-First Architecture:**
- All calendar/task data stays on user's self-hosted server by default
- Nothing leaves except explicit LLM queries
- Optional encrypted backup for users who want offsite redundancy

### LLM Data Handling

**Minimal & Configurable:**
- Minimal context by default - only send what's needed for the current planning request
- Strip names/details where possible; never send full calendar history
- User-configurable allowlist for data categories (event titles, attendee names, task details)
- Clear UI showing what will be sent before each LLM request (or on-demand audit)

### Self-Hosted Security

**Secure Defaults:**
- HTTPS required; no insecure fallbacks
- Strong authentication out of box
- Clear documentation of operator security responsibilities

**Security Tooling:**
- Auth logs and failed login visibility
- Session management
- API key rotation capability

### Data Ownership

**Full User Control:**
- Export all data (calendar, tasks, preferences, logs) in standard formats at any time
- Clean deletion - account removal deletes all associated data completely
- No orphaned data, no data retention after deletion

## Innovation & Novel Patterns

### Detected Innovation Areas

**Primary Innovation: Ambient Protection + Constraint-Guaranteed AI**
- **Ambient planning:** No ritual planning sessions; system continuously monitors calendar, detects interruptions, and proposes adaptations in real-time
- **Constraint-guaranteed AI:** LLM proposes, but hard constraints are deterministic and validated before display/apply - impossible to show or execute an invalid plan

**Secondary Innovation: Self-Hosted AI Planning**
- Unique combination of CalDAV ownership (data sovereignty, existing client compatibility) with AI planning layer
- First mover in "self-hosted calendar AI" - existing tools (Reclaim, Clockwise, Motion) are all cloud-first, vendor-locked

### Competitive Differentiation

| Calendar-app | Cloud Calendar AI Tools |
|--------------|------------------------|
| Self-hosted, data ownership | Vendor-locked, cloud-stored |
| Protection-first (sustainable pace) | Optimization-first (productivity metrics) |
| Works with existing CalDAV clients | Proprietary apps/integrations |
| Model-agnostic (swap LLMs) | Single AI provider locked |
| Constraint-guaranteed (no invalid plans) | AI-generated plans may violate constraints |

### Validation Approach

**Phase 1: Dogfooding (Weeks 1-4)**
- Creator (Daniel) uses Calendar-app daily
- Track: focus block completion rate, plan acceptance rate, sustainable pace indicators
- Success gate: daily use becomes indispensable; ambient protection noticeably reduces stress

**Phase 2: Small Cohort (Weeks 5-8)**
- 5-10 beta users from target persona (meeting-heavy knowledge workers, ADHD/context-switching sufferers)
- Track same metrics + qualitative feedback on protection philosophy
- Success gate: >60% retention, >70% focus block completion

### Risk Mitigation

**Graceful Degradation Architecture:**
- If AI planning quality is poor → CalDAV server still works as solid calendar
- If LLM provider fails → manual planning still works; blocks still protected
- Users never locked into broken AI; innovation layers are additive

**Manual Override Always Available:**
- Users can build their own plan, ignore AI suggestions
- System still protects explicitly-marked blocks
- Preference capture works even without AI proposals

## API Backend + Web App Specific Requirements

### Project-Type Overview

Calendar-app is a three-layer architecture:
- **CalDAV Server Layer:** CalDAV subset for interoperability; target conformance for the parts we implement
- **Web Application Layer:** Hybrid SPA with server-rendered shell and client-side reactivity for AI planning interface
- **REST API Layer:** Modern REST API for AI planning operations and integration endpoints

### Authentication & Authorization

**Hybrid Authentication Model:**
- **CalDAV clients:** HTTP Basic/Digest authentication over HTTPS (standard CalDAV auth that clients expect)
- **Web UI:** Modern session/token-based authentication
- **Shared user store:** Both authentication methods validate against the same user database
- **API keys:** Scoped read-only API keys for integration consumers (e.g., Slack bots)

**Authorization:**
- User-level permissions for calendar/task data access
- API key scoping (read-only vs. read-write, which endpoints allowed)
- Operator/admin role for deployment configuration and diagnostics

### Web Application Architecture

**Hybrid Rendering Strategy:**
- Server-rendered shell for initial page load speed and SEO
- Client-side reactivity for dynamic components (calendar grid, AI plan proposals, real-time updates)
- Core pages (login/settings/status) work with minimal JS; the interactive planning/calendar UI requires JS

**Real-Time Synchronization:**
- WebSocket connection for active sessions (instant updates when CalDAV changes detected)
- Polling fallback for compatibility and reconnection resilience
- Efficient diff-based updates to minimize bandwidth

**Browser Support Matrix:**
- **Desktop:** Latest 2 versions of Chrome/Firefox/Edge on Windows + Linux; Safari on macOS (latest 2)
- **Mobile:** iOS Safari (support ~2 years back); Android Chrome (current)
- **Excluded:** Internet Explorer; no legacy-specific requirements unless customer need emerges

### API Endpoints

**CalDAV Protocol:**
- CalDAV subset for interoperability (minimal conformant implementation)
- PROPFIND, REPORT, GET, PUT, DELETE for calendar operations
- WebDAV sync-collection for efficient incremental sync (optional/phase-2; evaluate for MVP)

**Internal REST API (MVP):**
Used by Calendar-app's own Web UI; not marketed as third-party API:
- `POST /api/v1/plan/daily` - Generate daily plan from calendar + Todoist state
- `POST /api/v1/plan/propose` - Propose plan adaptation (triggered by interruption)
- `POST /api/v1/plan/apply` - Apply user-confirmed plan changes to calendar/tasks
- `GET /api/v1/preferences` - Retrieve user preferences
- `PUT /api/v1/preferences` - Update user preferences (energy patterns, immovable tags, etc.)
- `POST /api/v1/todoist/sync` - Trigger Todoist sync operation

**Public Integration API (MVP-lite):**
Minimal read-only endpoints for simple external tools; scoped API keys required:
- `GET /api/v1/me/focus-status` - Query current user's focus block status (self-check for Slack bots, etc.)
- Focus blocks represented as calendar events (tagged/dedicated calendar); API derives status from CalDAV state
- Teammate queries (`/api/v1/users/{id}/focus-status`) require explicit opt-in + permission grant; can be deferred to Post-MVP if needed

### Rate Limiting

**Layered Protection:**
- **Per-user global limit:** 100 requests/minute per authenticated user (prevents abuse)
- **Per-endpoint specific limits:**
  - CalDAV operations: generous/default-high limits + operator-configurable; protect against abuse without breaking normal client sync
  - AI planning endpoints: 10 requests/minute (protects expensive LLM calls)
  - Integration API: 60 requests/minute (reasonable polling rate)

**Response Headers:**
- RateLimit / X-RateLimit headers (conventional; subject to change)
- HTTP 429 (Too Many Requests) with Retry-After header

### API Versioning

**URL-based Versioning:**
- `/api/v1/` for initial API version
- Future versions at `/api/v2/`, etc.
- MVP starts with v1; don't over-engineer for hypothetical v2
- Breaking changes require new version; additive changes OK in existing version

### Data Formats

**Request/Response:**
- JSON for REST API (planning, integration endpoints)
- XML (WebDAV/CalDAV) + iCalendar (.ics) for CalDAV operations (RFC 5545)
- Standard HTTP status codes (200, 201, 400, 401, 403, 404, 409, 429, 500)

**Error Responses:**
```json
{
  "error": "constraint_violation",
  "message": "Proposed plan overlaps with immovable CEO meeting",
  "details": { "conflicting_event_id": "abc123" }
}
```

### Accessibility

- WCAG 2.1 AA compliance for web UI
- Keyboard navigation for all interactive elements
- Screen reader compatibility
- Focus indicators for keyboard users

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-Solving + Experience MVP
- Validate that ambient protection + constraint-guaranteed AI solves burnout/fragmentation for knowledge workers
- Validate that self-hosted CalDAV ownership + data sovereignty delivers compelling user experience
- Both together prove the unique value proposition

**Resource Requirements:**
- Solo developer (Daniel) initially - full-stack ownership
- Grow team based on validation results (Weeks 5-8 cohort feedback)
- Skills needed: Backend (Go/Rust/Python), CalDAV/WebDAV protocols, Frontend (React/Vue/Svelte), AI integration, DevOps (Docker, nginx)

**Development Approach:**
- Build core infrastructure first (CalDAV + constraint validation), test with all 3 clients before adding AI
- Daily dogfooding exposes issues early; iterate rapidly on single-user experience
- Speed to validation over feature completeness

### MVP Feature Set (Phase 1)

**Core User Journeys Supported:**
1. Marcus - Meeting-heavy executive (AI finds focus time in wall-to-wall calendar)
2. Priya - Deep-work creator (AI protects large uninterrupted blocks)
3. Dana - Self-hoster/operator (deployment, multi-client sync, diagnostics)
4. Edge case: Sync conflicts (multi-client concurrent edits handled gracefully)
5. Edge case: AI rejection loop (preference capture when plans fail repeatedly)

**Must-Have Capabilities:**

| Capability | MVP Scope |
|------------|-----------|
| **CalDAV Server** | Minimal interoperable subset: CRUD + basic sync with Fantastical, Apple Calendar, DAVx⁵; basic recurring events (RRULE support); conflict detection and resolution |
| **Todoist Integration** | Read tasks/projects; write updates (priority, due dates, labels); link to calendar time blocks; bidirectional sync |
| **AI Planning Engine** | Daily plan generation; interruption detection; adaptation proposals; <2s latency; hard-constraint validation (deterministic, runs before display/apply) |
| **Focus Block Protection** | Detect calendar interruptions; propose options (split/defend/absorb/negotiate); track completion; protection level configuration |
| **Preference System** | Store/retrieve user rules (energy patterns, minimum blocks, post-meeting task types, immovable attendee/event tags) |
| **Propose → Confirm → Apply Loop** | Draft reschedule messages; apply changes after user confirmation; respect organizer permissions; handle CalDAV sync flow |
| **Conflict Resolution UI** | Surface sync conflicts; show diff with timestamps; user decides winner; log resolution; re-propose plan after resolution |
| **Operator Tooling** | Env/config file deployment; sync logs with client fingerprints; debug bundle export; backup-friendly data directory |
| **Web UI** | Hybrid rendering; calendar grid view; plan proposal interface; WebSocket + polling for real-time updates; preference configuration UI |
| **Internal REST API** | Planning endpoints, preferences, Todoist sync - used by Web UI (not marketed as third-party API) |
| **Public Integration API (MVP-lite)** | Minimal read-only: `GET /api/v1/me/focus-status` for self-check (enables simple Slack bot); scoped API keys |

**Explicitly Deferred to Post-MVP:**
- Full third-party integration API ecosystem (team queries, expanded endpoints, SDKs, developer docs) - though minimal self-check endpoint included in MVP
- Multi-calendar unified view (work + personal)
- Team features (shared calendars, multi-user coordination)
- Advanced CalDAV (sync-collection, complex recurrence edge cases, full conflict coverage)
- Additional task manager integrations beyond Todoist
- Predictive intelligence (pattern learning, preemptive blocking)

### Post-MVP Features

**Phase 2 (Growth):**
- Full integration API platform (team queries with opt-in, richer endpoints, webhooks, SDKs, developer documentation)
- Multi-calendar intelligence (work + personal calendars unified view)
- Integration expansion beyond Todoist (Asana, Linear, ClickUp, etc.)
- Advanced CalDAV features (sync-collection, WebDAV sync tokens, complex recurrence rules)
- Mobile-responsive UI optimization

**Phase 3 (Expansion):**
- Team orchestration (shared calendars, cross-team availability, multi-user planning coordination)
- Predictive intelligence (learns user patterns, preemptive time blocking, burnout prediction/warnings)
- Integration hub ecosystem (email integration, note apps, broader productivity ecosystem)
- Multi-tenant SaaS offering (optional hosted version alongside self-hosted)

### Risk Mitigation Strategy

**Technical Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| **CalDAV multi-client sync reliability** | High - data loss destroys trust | Build CalDAV first; test extensively with all 3 clients before adding AI; daily dogfooding catches sync issues; implement conflict detection from day 1 |
| **Constraint validation correctness** | High - invalid plans break "constraint-guaranteed" promise | Implement deterministic constraint checker separately from LLM; unit test all constraint scenarios; dogfood validates edge cases |
| **LLM API reliability/cost** | Medium - external dependency | Model-agnostic harness from day 1; graceful degradation if LLM fails; manual override always available; use cheaper models (Grok) during development |
| **Recurring event complexity** | Medium - even basic RRULE is non-trivial | Scope to simple daily/weekly recurrence for MVP; defer complex patterns (nth weekday, exceptions) to Phase 2 |

**Market Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| **"Self-hosted is too hard"** | High - target users bounce on setup | Dana's journey proves deployment; if dogfooding reveals friction, simplify installer before cohort; provide 1-click Docker Compose setup |
| **"AI planning doesn't beat manual"** | High - core value prop fails | Track plan acceptance rate during dogfooding; <80% acceptance = AI needs improvement before cohort; iterate on prompt engineering and constraint tuning |
| **Cloud competitors add self-hosted** | Medium - differentiation weakens | Speed to MVP; protection-first philosophy is harder to copy than self-hosting; model-agnostic harness as additional moat |

**Resource Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Solo dev takes longer than expected** | Medium - delayed validation | Cut basic recurring events from MVP (single events only); defer Web UI polish; prioritize core loop validation over UX refinement |
| **Dogfooding reveals scope creep** | Medium - MVP bloat | Ruthless scope discipline: if feature not in Marcus/Priya/Dana journeys, defer to Phase 2; validate learning > feature completeness |
| **Can't reach retention targets in cohort** | High - product-market fit question | Graceful pivot: CalDAV server still valuable standalone; AI becomes "experimental beta feature"; learn what users value most |

## Functional Requirements

### Calendar Management (CalDAV)

- **FR1:** Users can create calendar events with title, start/end time, description, location
- **FR2:** Users can edit existing calendar events (time, details, recurrence)
- **FR3:** Users can delete calendar events
- **FR4:** Users can create recurring events with basic patterns (daily, weekly)
- **FR5:** Users can sync their calendar with external CalDAV clients (Fantastical, Apple Calendar, DAVx⁵)
- **FR6:** The system can detect when calendar events are created/modified/deleted by external CalDAV clients
- **FR7:** The system can detect sync conflicts when the same event is edited simultaneously from multiple clients
- **FR8:** Users can resolve sync conflicts by choosing which version to keep after reviewing a diff

### Task Management (Todoist Integration)

- **FR9:** Users can connect their Todoist account to Calendar-app
- **FR10:** The system can read tasks and projects from the user's Todoist account
- **FR11:** The system can write updates back to Todoist (priority, due dates, labels/sections/comments)
- **FR12:** The system can link Todoist tasks to planned calendar time blocks
- **FR13:** Users can trigger manual Todoist sync operations

### AI Planning & Scheduling

- **FR14:** Users can request a daily plan generated from their calendar and Todoist state
- **FR15:** The system can propose time blocks for high-priority tasks based on available calendar time
- **FR16:** The system can detect when new calendar events interrupt existing focus blocks
- **FR17:** The system can propose plan adaptations when interruptions occur (split, defend, absorb, negotiate)
- **FR18:** Users can review AI-proposed plans before accepting or modifying them
- **FR19:** Users can confirm AI-proposed plan changes with 1-click apply
- **FR20:** The system can apply confirmed plan changes to both calendar (new time blocks) and Todoist (updated metadata)
- **FR21:** The system can draft reschedule messages for meetings that need to move
- **FR22:** The system validates all proposed plans against hard constraints before display (no overlaps with fixed events, respects allowed time windows)

### Focus Protection & Interruption Management

- **FR23:** Users can configure focus time preferences (minimum block duration, preferred time windows)
- **FR24:** Users can mark specific calendar events or attendees as "immovable" (high-priority, cannot be rescheduled)
- **FR25:** The system can detect when a new meeting conflicts with a protected focus block
- **FR26:** Users can choose how to handle focus block interruptions (split, protect, negotiate, sacrifice)
- **FR27:** The system can draft async-alternative messages (proposing Loom/Slack instead of sync meetings)
- **FR28:** Users can track focus block completion status
- **FR29:** The system can mark focus blocks as completed when no conflicts occurred during scheduled time

### User Preferences & Learning

- **FR30:** Users can configure energy pattern preferences (peak mental energy times, post-meeting task types)
- **FR31:** Users can set minimum focus block durations
- **FR32:** Users can configure meeting batching preferences
- **FR33:** The system can store user-defined constraint rules for future plan generation
- **FR34:** Users can trigger guided preference capture when AI plans repeatedly fail
- **FR35:** The system can ask targeted questions to discover user work patterns (energy levels, task priorities, time-of-day preferences)

### Data Privacy & Security

- **FR36:** Users can configure which data categories are sent to LLM providers (event titles, attendee names, task details)
- **FR37:** Users can audit what data will be sent before each LLM request
- **FR38:** Users can export all their data (calendar, tasks, preferences, logs) in standard formats
- **FR39:** Users can delete their account and all associated data completely

### Operator Administration

- **FR40:** Operators can deploy Calendar-app via Docker Compose or binary with env/config file
- **FR41:** Operators can configure CalDAV server settings (storage path, domain/port, auth method)
- **FR42:** Operators can view sync logs with client fingerprints for troubleshooting
- **FR43:** Operators can export debug bundles for issue investigation
- **FR44:** Operators can configure CalDAV rate limits
- **FR45:** Operators can view authentication logs and failed login attempts
- **FR46:** Operators can manage API keys (create, revoke, rotate)
- **FR47:** Operators can back up the calendar data directory

### Integration & Extensibility

- **FR48:** Users can generate scoped API keys for third-party integrations
- **FR49:** External tools can query a user's current focus block status (self-check only)
- **FR50:** The system exposes a read-only integration API deriving status from CalDAV state (no separate data store)

### Authentication & User Management

- **FR51:** Users can log in to the web UI using their credentials
- **FR52:** Users can log out of the web UI
- **FR53:** Users can manage CalDAV credentials used by external CalDAV clients
- **FR54:** Users can generate app-specific passwords for CalDAV client authentication
- **FR55:** Users can rotate or reset their authentication credentials

### Calendar & Task Views

- **FR56:** Users can view a unified day agenda showing calendar events, focus blocks, and Todoist tasks together
- **FR57:** Users can switch between day and week views
- **FR58:** Users can navigate to specific dates (past and future)

### Audit Log & Rollback

- **FR59:** The system records an audit log for all plan apply actions (what changed, when, which source/trigger)
- **FR60:** Users can view the audit log of recent plan changes
- **FR61:** Users can undo/rollback the last applied plan bundle within a defined time window
- **FR62:** The system can attempt to restore calendar and Todoist state to pre-apply conditions when rollback is triggered; if conflicts exist due to external changes (e.g., CalDAV client edits after apply), the system surfaces the conflicts and requires user choice to complete rollback

### Capacity & Overload Management

- **FR63:** The system can detect when planned work exceeds available time (overload condition)
- **FR64:** The system can surface overload warnings to users before displaying proposed plans
- **FR65:** The system can propose tradeoffs when overload is detected (defer tasks, reduce scope, extend timeline)

## Non-Functional Requirements

### Performance

**Response Time Targets:**
- Plan generation: <2 seconds from trigger to display (per success criteria)
- CalDAV sync operations: <500ms for CRUD operations
- Web UI interactive elements: <100ms feedback for user actions (drag events, click apply, UI updates)
- Real-time update propagation: <1 second via WebSocket

**Background Efficiency:**
- CalDAV polling and Todoist sync operations execute without blocking or slowing the Web UI
- Background sync processes consume minimal CPU/memory when idle

### Reliability

**Data Integrity:**
- Zero data loss across all operations (CalDAV sync, plan apply, rollback, conflict resolution)
- All state changes are atomic or reversible
- Audit log maintains complete history of plan modifications

**Graceful Degradation:**
- When LLM API fails: manual planning still works; existing preferences still applied; blocks still protected
- When Todoist API fails: core calendar functions continue; sync retries when connectivity restored
- When WebSocket fails: polling fallback activates automatically

**Availability:**
- Self-hosted deployment; uptime responsibility rests with operator
- System provides clear error states and recovery paths when external dependencies fail

### Security

**Data Encryption:**
- Encryption at rest supported via operator-managed disk/filesystem encryption (recommended) or optional app-level encryption (configurable)
- All network communication uses TLS 1.2+ (HTTPS required; no insecure fallbacks)
- LLM API requests use encrypted connections

**Authentication & Session Security:**
- Session tokens expire after configurable inactivity period (default: 24 hours)
- API keys are revocable and rotatable
- Rate limiting prevents brute-force authentication attempts
- Failed login attempts are logged and visible to operators

**Access Control:**
- User-level data isolation (single-user MVP; no cross-user access)
- API key scoping enforced (read-only vs read-write, endpoint restrictions)
- Operator role separation for deployment configuration

### Integration Reliability

**Fault Tolerance:**
- Todoist API failures handled with bounded/configurable exponential backoff with max retry window; user-visible failure state + manual retry
- LLM API failures degrade gracefully to manual planning mode
- System displays clear error states and allows manual retry

**Offline Resilience:**
- Core calendar functions (view, create, edit, delete events) work when Todoist/LLM are unreachable
- Pending sync operations queue and execute when connectivity restored
- User notified of offline state; no silent failures

### Scalability

**Single-User MVP Targets:**
- Single instance supports one user with multiple concurrent sessions/devices (web UI + 3 CalDAV clients) without degradation
- CalDAV server handles concurrent sync requests from multiple clients for the same user
- System resource usage remains stable under typical single-user load

**Future Multi-User Considerations:**
- Architecture designed for horizontal scaling (multi-tenant SaaS deferred to Phase 3)
- Database and file storage approach supports future optimization
- No hard-coded single-user assumptions that would block multi-user expansion

### Accessibility

**WCAG 2.1 AA Compliance:**
- Keyboard navigation for all interactive elements
- Screen reader compatibility with semantic HTML and ARIA labels
- Focus indicators visible for keyboard users
- Color contrast meets AA standards
- Forms and error messages accessible
