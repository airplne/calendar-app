---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: []
session_topic: 'Calendar-app: model-agnostic, chat-first calendar + task assistant with CalDAV server + Todoist'
session_goals: 'Problem statement, MVP vs v1 features, integration requirements, killer use-cases, product/UX concepts'
selected_approach: 'AI-Recommended Techniques'
techniques_used: ['Question Storming', 'Persona Journey', 'First Principles Thinking']
ideas_generated: ['MVP Core Loop', 'Protection > Optimization', 'Ambient Planning', '4 Persona Use-Cases', '7 First Principles Differentiators']
context_file: ''
workflow_completed: true
---

# Brainstorming Session Results

**Facilitator:** Daniel
**Date:** 2026-01-15

## Session Overview

**Topic:** Calendar-app — a model-agnostic, chat-first calendar + task assistant that runs a first-class CalDAV server (we become the calendar provider) plus Todoist integration, enabling users to chat with any LLM (Grok, Claude, Gemini, OpenAI, etc.) which can pull full context and take actions.

**Goals:**
1. Crisp problem statement + target user workflow
2. MVP vs v1 feature breakdown
3. Integration requirements/constraints (auth, scopes, caching, privacy)
4. Killer differentiating use-cases
5. Product/UX concepts (chat-first, context transparency, confirm flow, audit/undo)

### Problem Statement

**The pain:** Every planning decision requires mentally holding calendar + task + priority context, synthesizing it yourself, then manually executing changes. When reality shifts (new meeting), the whole plan collapses and requires manual re-planning.

**The vision:** A chat-first assistant that *already knows* your calendar and tasks, reasons about priorities and constraints, proposes a plan, and — with your confirmation — executes changes directly. The AI becomes a planning co-pilot, not a context-starved tool.

### Architecture Summary

| Layer | Design |
|-------|--------|
| **Backend** | MCP-style context service exposing calendar + task data to any AI client |
| **Frontend** | Standalone app with built-in chat UI, model-agnostic LLM backends (Grok/Claude/Gemini/OpenAI/etc.) |
| **Integrations** | CalDAV server (clients: Fantastical/Apple Calendar/Android CalDAV clients/etc. sync to us); Todoist REST API |
| **Context** | Local assembly/cache/summaries → minimal context to cloud LLM; user controls sharing |
| **Trust** | Confirm-by-default, escalation to auto-apply for low-risk, audit log + undo |
| **Platform** | Linux-first, then macOS/Windows; mobile via responsive web |
| **License** | Open-source core, bring-your-own-keys |

### MVP Exit Criteria

The loop works end-to-end:
1. **Read:** "Show my day with task priorities overlaid on calendar"
2. **Write:** "Create a time block for [task X]"
3. **Replan:** "Reschedule my afternoon around this new meeting"

All via propose → confirm → apply, with real updates to our CalDAV server + Todoist.

---

## Technique Selection

**Approach:** AI-Recommended Techniques
**Analysis Context:** Calendar-app product definition with focus on problem statement, MVP/v1 features, integration requirements, killer use-cases, and UX concepts

**Recommended Techniques:**

1. **Question Storming** (deep): Ensure we're solving the right problem by generating questions before answers — sharpens problem framing and surfaces hidden assumptions
2. **Persona Journey** (theatrical): Embody target users to generate killer use-cases and UX requirements organically through lived experience
3. **First Principles Thinking** (creative): Strip away assumptions about how calendar apps "should" work and rebuild from fundamental truths for differentiation

**AI Rationale:** These three techniques form a logical arc: Question Storming ensures problem clarity, Persona Journey generates grounded use-cases, and First Principles drives innovation and MVP scoping. Optional extensions (Constraint Mapping, Cross-Pollination) available for integration requirements and UX innovation.

---

## Technique 1: Question Storming Results

### Core Design Questions Resolved

**Planning Layers & Atomic Units:**
- AI chooses working layer (goals → tasks → blocks → energy) based on intent + confidence; user can override
- Default flow: tasks → proposed time blocks with buffers/energy awareness
- "Too vague" gate: if task lacks next action, success criteria, or duration → ask clarifiers or propose breakdown
- Block ends but task isn't done: never silently reshuffle; prompt with options (extend, reschedule, split)

**Ask vs. Infer (Risk-Based):**
- Ask when: high-impact, low-confidence, preference-sensitive
- Infer when: low-risk, reversible; clearly state assumptions
- Reduce fatigue via batching, defaults, "assume for this session" toggles

**Trust & Write Authority:**
- MVP: propose → confirm → apply (no auto-apply)
- Later: opt-in autopilot for trivial reversible changes only (shift personal block ±10min, no attendees, no deletions)
- Always confirm: attendees, deletions, major moves

**Learning & Intelligence:**
- High-leverage vs busywork: user sets top goals/labels; system learns locally from feedback
- Chronic deferrals: detect repeated punts → surface decision (delete/delegate/breakdown/schedule)
- Priority vs behavior conflicts: surface mismatches, propose protected blocks

**UX Patterns:**
- Diff + reasoning: 3-5 headline changes + counts, expandable grouped diff, simple day timeline
- Reasoning: 3 bullets + key constraints; "why?" expands on demand
- Undo/audit: durable change log (7-30 days+), per-action undo, "rollback last apply bundle"

**Multi-Model & Hallucination Defense:**
- Multiple models propose; deterministic constraint checker validates against real data
- Present top 1-2 validated plans; model attribution hidden by default
- Never trust model-stated constraints; validate against source-of-truth before showing/writing

**Privacy & Multi-User:**
- Local context assembly/cache/summaries (optionally embeddings)
- Cloud LLMs receive minimal redacted "context pack" (free/busy, windows, task metadata)
- Learning/preferences stored local-first, exportable
- Team-ready: strict per-user/tenant isolation; shared org knowledge explicit + permissioned

### Uncomfortable Questions — Consolidated Answers

| # | Question | Answer |
|---|----------|--------|
| 24 | People don't want AI planning? | Design "AI suggests, human always controls" — satisfy both camps |
| 25 | Real competitor = learned helplessness? | Target users already trying to change (productivity enthusiasts) |
| 26 | LLM costs stay high? | Optimize for minimal tokens + design for cheap/local models as fallback |
| 27 | Why hasn't someone won? | They have (Reclaim, Motion) — we're doing it open + cheaper |
| 28 | Day 1 value with zero learning? | All of: read-only value first, defaults, import, set expectations |
| 29 | Minimum viable context? | Need both calendar AND tasks — either alone is weak |
| 30 | Chaotic calendar handling? | Offer optional "calendar audit" mode as first step |
| 31 | AI makes bad call — who's blamed? | Audit log makes it clear what happened and why |
| 32 | Prevent over-trusting AI? | Show reasoning + confidence scores + make diffs interesting + occasional "are you sure?" |
| 33 | Context pack subtly wrong? | Defense in depth: validate against source, show context summary, deterministic checker |
| 34 | Open-source fork risk? | Nothing stops them — and that's fine; building for ourselves and community |
| 35 | Model-agnostic = no defensibility? | Feature — users want choice and hate vendor lock-in |
| 36 | What can't be copied? | Nothing — everything can be copied; we compete on execution |
| 37 | Building because best or enjoyable? | Both — and that's the sweet spot |
| 38 | What makes us abandon in 6mo? | All real risks: better alternative, complexity, life shifts |
| 39 | Build if competitors go free? | Yes — they're still closed/locked-in; open-source matters |

### Key Insights from Question Storming

1. **Execution > IP** — No moat from features; compete on taste, speed, community
2. **Trust is earned** — MVP is confirm-only; autopilot comes later with track record
3. **Both calendar AND tasks required** — Either alone doesn't hit value threshold
4. **Target changers** — Don't try to convert the skeptical; find users already in motion
5. **Defense in depth** — Never trust AI alone; validate everything against source-of-truth
6. **Open-source is the differentiator** — Not despite competition, but because of it

---

## Technique 2: Persona Journey Results

### Persona 1: Jordan (The Aspiring Optimizer)

**Profile:** Late 20s/early 30s knowledge worker. Has read GTD, tried Notion/Todoist/timeboxing. Nothing sticks longer than 2 weeks.

**Core Pain:** Systems don't stick; planning paralysis; guilt when things go wrong.

**Killer Use-Cases:**
| Use-Case | User Story |
|----------|------------|
| Instant Planning | "Plan my day" → 10 seconds → Apply → Done |
| Graceful Recovery | Detects drift, acknowledges without guilt, adapts forward |
| Focus Protection | Suggests alternative times for interruptions, tracks patterns |
| Chronic Punt Surfacing | "You've moved this 6 times — delete, delegate, or commit?" |
| Pattern Learning | "You lose focus at 2pm — lighter tasks there?" |
| End-of-Day Closure | Shows what you *did* accomplish, moves the rest |

**Magic Moment:** "I typed 'plan my day' and it just... did. 10 seconds. I hit apply."

**Retention Driver:** Speed + recovery + no guilt + visible learning.

---

### Persona 2: Sam (The Overloaded IC)

**Profile:** Senior engineer, 6-8 meetings/day, deep work constantly fragmented, ADHD tendencies.

**Core Pain:** No time for actual work; meetings fragment the day; context-switching is 10x expensive.

**Killer Use-Cases:**
| Use-Case | User Story |
|----------|------------|
| Overload Quantification | "32 hrs meetings / 20 hr target. Here's what could move." |
| Work-Time Math | "14 hours committed, 6 available. Something has to give." — BEFORE committing more |
| Prep Time Injection | Detects meetings needing prep, proposes time automatically |
| Focus Quality Awareness | Pre-interview focus ≠ open morning focus; suggests accordingly |
| Invasion Pattern Tracking | "Focus blocks interrupted 4x this week. Here's the pattern." |
| ADHD-Friendly Batching | Groups related tasks, minimizes transitions, learns switch costs |
| Manager Advocacy Data | Exportable "here's why I'm overloaded" with real numbers |

**Magic Moment:** "It showed me I'm at 32 hours of meetings against a 20-hour target. With data."

**Retention Driver:** Made the invisible visible + advocacy support.

---

### Persona 3: Alex (The Solo Consultant)

**Profile:** Mid-30s freelancer, 4-5 active clients, each with own calendar/Slack/priorities. Every client thinks they're the only client.

**Core Pain:** Multi-calendar chaos; no hours visibility; time tracking always retroactive; constant diplomacy.

**Killer Use-Cases:**
| Use-Case | User Story |
|----------|------------|
| Multi-Calendar Conflict Resolution | Detect cross-client conflicts, propose diplomatic reschedule with no leak |
| Retainer Hours Warning | "14/15 hours Client A. 1 hour left." — BEFORE going over |
| Client Time Dashboard | Monday view: hours per client vs. targets, admin flagged if zero |
| Context Switch Briefs | Opt-in 2-min brief before switching clients, pulled from real notes |
| Energy-Aware Gap Filling | "2 hr gap, you're fresh, Client B due Friday — start now?" |
| Tiered Availability | Different response SLAs per client + auto-protect with fake "meetings" |
| Realistic Personal Time | Don't protect fiction; reschedule to slots that will actually happen |

**Magic Moment:** "It told me I'm at 14 of 15 hours with Client A before I went over."

**Retention Driver:** Stopped over-delivering + diplomatic conflict handling.

---

### Persona 4: Morgan (The Manager Without Admin)

**Profile:** Late 30s Engineering Manager, 12 direct reports, no EA, 9 1:1s/day, zero time for IC work.

**Core Pain:** Calendar is 90% meetings; can't tell which 1:1s can move; "manager" became "meeting attendee."

**Killer Use-Cases:**
| Use-Case | User Story |
|----------|------------|
| 1:1 Moveability Scoring | Know which 1:1s are safe to move vs. high-touch (notes + manual tags) |
| Calendar Tetris Solver | "No room" → show trade-offs + suggest lowest-cost + push back to requestor |
| 1:1 Batching | Proactively suggest consolidating check-in 1:1s to free IC time |
| Meeting Load Benchmarking | "38 hrs meetings vs. 28 hr peer average. Here's what could change." |
| Leadership Visibility | Opt-in sharing of trade-offs with director as conversation starter |
| Drift Detection | Flag when I've become "meeting attendee" instead of "manager" |

**Magic Moment:** "It showed me I'm at 38 meeting hours vs. 28 peer average — and what could change."

**Retention Driver:** Diagnosed drift + prescribed solutions + leadership visibility.

---

### Cross-Persona Insights

| Dimension | Jordan | Sam | Alex | Morgan |
|-----------|--------|-----|------|--------|
| **Core pain** | Systems don't stick | No time for work | Multi-client chaos | Calendar Tetris |
| **Trust trigger** | Speed + no guilt | Data + advocacy | Diplomacy + hours | Trade-offs + benchmarks |
| **Magic moment** | 10-sec planning | Meeting load data | Hours warning | Peer comparison |
| **Unique need** | Graceful recovery | ADHD-friendly batching | Client conflict handling | 1:1 moveability |

---

## Technique 3: First Principles Results

### Assumptions Challenged

| # | Assumption | Answer | Design Implication |
|---|------------|--------|-------------------|
| 61 | Calendar is the right abstraction | CB | Calendar for coordination; personal planning needs different model + implementation is wrong, not abstraction |
| 62 | Planning should be separate activity | CB | Ambient planning + upfront with continuous adaptation |
| 63 | Optimization is the goal | CB | Protection > optimization + selective for high-leverage only |
| 64 | AI needs to "understand" you | CB | Learn patterns with confirmation + allow override |
| 65 | Unified view is better | CB | Unified context matters (UI flexible) + tools not talking is the problem |
| 66 | Tool should be invisible or opinionated | CB | Opinionated about protection, invisible about method + opt-in coach mode |
| 67 | What's the atomic unit of work | CB | Commitments are hard constraints + tasks fill the rest |
| 68 | Who is system accountable to | CB | Surface conflicts + escalate friction based on stakes |

### Key Differentiators from First Principles

1. **Calendar is for coordination, not personal planning** — Don't force the grid; let tasks/outcomes drive, calendar receives
2. **Ambient planning > ritual planning** — No "sit down and plan" requirement; tool adapts continuously
3. **Protection > optimization** — Guard time from low-leverage; don't help pack more in
4. **Unified context, flexible UI** — The AI sees everything; how you view it is your choice
5. **Opinionated about protection, invisible about method** — Strong defaults on guarding time; no judgment on how you work
6. **Commitments = hard constraints; tasks = flexible** — Other people's time is sacred; your own is negotiable
7. **Surface conflicts, escalate by stakes** — Show the gap between stated/actual; push harder on what matters

---

## Idea Organization & Prioritization

### MVP (Core Loop)

| Feature | What It Does |
|---------|--------------|
| Day View | Tasks + calendar in one view |
| Propose → Confirm → Apply | AI suggests, you approve, it writes |
| Conflict Detection | Catch scheduling clashes early |
| Overload Warning | "More work than time" alert |
| Time Block Creation | Schedule tasks with one command |
| Graceful Recovery | Adapt when plans derail |

### v1 Enhancements

| Feature | What It Does |
|---------|--------------|
| Pattern Learning | "You always lose focus at 2pm" |
| Meeting Load Benchmarks | Compare to peers/targets |
| Multi-Client Handling | Conflict diplomacy + hours tracking |
| 1:1 Moveability | Which meetings are safe to move |
| ADHD Batching | Group related tasks |
| Advocacy Export | Data to show your manager |

---

## Action Plan — MVP Implementation

### Immediate Next Steps

1. **Build CalDAV server** — First-class CalDAV endpoints; our app becomes calendar source of truth
2. **Set up Todoist integration** — Todoist REST API for tasks
3. **Build context service** — Local cache/assembly of calendar + task data
4. **Create day view** — Unified display: tasks overlaid on calendar
5. **Implement propose → confirm → apply** — AI generates plan, user approves, system writes
6. **Add conflict detection** — Surface clashes before they happen
7. **Deploy overload warning** — "X hours work, Y hours available"

### Architecture Decisions Locked

- CalDAV server as calendar source of truth (Fantastical/Apple Calendar/Android clients sync to us)
- MCP-style backend (any AI client can call)
- Local context assembly → minimal to cloud LLM
- Confirm-by-default (no autopilot in MVP)
- Todoist REST API for tasks
- Per-user isolation from day 1 (single-user MVP, multi-user ready)

### Key Differentiators

- Protection > optimization
- Ambient planning (no ritual)
- Open-source, bring-your-own-keys
- Unified context, flexible UI

---

## CalDAV Server Requirement (Core Integration Decision)

We will implement a first-class CalDAV server as part of Calendar-app so our app can act as a calendar "source of truth" that any client (Fantastical, Apple Calendar, Android CalDAV clients, Thunderbird, etc.) can subscribe to with full read/write sync. This avoids dependence on Fantastical-specific APIs and enables cross-platform compatibility.

### Why CalDAV (vs only REST)

- Fantastical and many calendar clients sync via CalDAV; Fantastical has no robust public REST API
- Enables Linux-first + macOS/Windows + iOS/Android interoperability
- Lets our app be the unified calendar backend while remaining open-source and provider-agnostic

### Implementation Notes / Acceptance Criteria

- Provide CalDAV endpoints for calendars + events (create/read/update/delete), supporting recurring events, time zones, and ETags/sync tokens
- Authentication + per-user isolation from day 1 (single-user MVP, multi-user ready)
- Server must interoperate with Fantastical and at least one Android CalDAV client in MVP validation
- Keep propose → confirm → apply on writes initiated by the AI; direct client edits still sync and are logged/auditable

### Open Questions (Resolve in Architecture/PRD)

- Single-tenant local-first vs hosted multi-tenant deployment model
- Storage format (e.g., ICS on disk vs DB) and conflict-resolution strategy
- Whether we also expose a REST API/MCP service on top of CalDAV for AI/planning operations

---

## Session Summary

**Techniques Used:** Question Storming, Persona Journey (4 personas), First Principles Thinking

**Key Outcomes:**
- Crisp problem statement: AI planning co-pilot that already knows your context
- 4 validated personas with distinct pain points and killer use-cases
- MVP/v1 feature split with clear prioritization
- 7 key differentiators from first principles
- Actionable implementation roadmap

**Core Insight:** Ambient planning IS protection. The tool's job is to guard your time from low-leverage work while adapting continuously — no planning ritual required.

---
