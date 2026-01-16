# PRP: Research — Calendar-app starters + monorepo layout

**Type:** Research Prompt
**Target Agent:** GPT-5 Pro
**Date:** 2026-01-16
**Author:** Daniel

---

## 1. Context

**Calendar-app** is an AI planning co-pilot built on a self-hosted CalDAV server. The technical stack decisions are:

| Layer | Technology | Notes |
|-------|------------|-------|
| **Backend** | Go | CalDAV/WebDAV protocol server + REST APIs |
| **Frontend** | React | Framework TBD (Next.js vs Vite); Tailwind + headless primitives |
| **Database** | SQLite | MVP single-user; schema designed for future Postgres migration |
| **Deployment** | Docker + single binary | Linux-first; self-hosted |

**Architectural priorities:**
- Minimal lock-in; avoid heavy full-stack starters
- Focus on interfaces, data model, and failure modes before implementation details
- CalDAV subset minimal; prioritize multi-client sync correctness (Fantastical, Apple Calendar, DAVx⁵)
- Graceful degradation: calendar must work if Todoist/LLM services are down
- Propose → Confirm → Apply interaction model; no silent writes

**Key references:**
- PRD: `_bmad-output/planning-artifacts/prd.md` (65 FRs, 6 NFR categories)
- UX Spec: `_bmad-output/planning-artifacts/ux-design-specification.md`
- Architecture (in progress): `_bmad-output/planning-artifacts/architecture.md`

---

## 2. Goals / Questions GPT-5 Pro Must Answer

### 2.1 Go Backend Starter Patterns

Recommend minimal Go backend starter patterns for:

| Area | Questions |
|------|-----------|
| **CalDAV/WebDAV endpoints** | Use `emersion/go-webdav` or build from scratch? How to structure handlers? |
| **REST APIs** | Chi vs Gin vs Echo vs stdlib? OpenAPI-first or code-first? |
| **Background jobs** | Embedded scheduler (cron-like) or separate worker? Job queue patterns? |
| **Configuration** | Viper vs envconfig vs stdlib? Config file format (YAML/TOML/env)? |
| **Logging/telemetry** | Structured logging (slog vs zerolog vs zap)? Metrics? Tracing? |
| **Database/migrations** | SQLite driver choice? Migration tool (goose, golang-migrate, atlas)? |
| **Test strategy** | Table-driven tests? Integration test patterns for CalDAV? Mocking LLM/Todoist? |

### 2.2 React Web Starter Approach

Recommend minimal React web starter aligned with UX spec:

| Area | Questions |
|------|-----------|
| **Build tool** | Next.js vs Vite — compare for our use case (hybrid SPA, SSR shell) |
| **Styling** | Tailwind CSS setup; headless primitives (Radix, Headless UI, Ark UI) |
| **Component ownership** | Copy-paste pattern (shadcn/ui style) vs npm dependencies |
| **State management** | Zustand vs Jotai vs TanStack Query only? |
| **Real-time** | WebSocket/SSE client patterns; reconnection; optimistic updates |
| **Form handling** | React Hook Form vs native? Validation patterns? |

### 2.3 Monorepo Layout

Recommend a monorepo structure:

```
Calendar-app/
├── server/          # Go backend
├── web/             # React frontend
├── docs/            # Documentation
├── scripts/         # Dev/build/deploy scripts
└── ...
```

Questions:
- Package manager: Go workspaces? pnpm workspaces? Turborepo? Nx? Keep simple?
- Dev workflow: One command to start both server + web with hot reload?
- Shared types: How to share API types between Go and TypeScript?
- Build artifacts: Where do compiled assets live?
- CI shape: GitHub Actions structure for lint/test/build/release?

### 2.4 Packaging & Deployment

Recommend packaging and deployment strategy:

| Area | Questions |
|------|-----------|
| **Single binary** | Embed web assets in Go binary (embed.FS)? Separate artifacts? |
| **Docker images** | Multi-stage build? Base image (distroless, alpine, scratch)? |
| **Config/env** | Env vars vs config file? Defaults? Secret handling patterns? |
| **Data directory** | SQLite location? Backup-friendly layout? Data export format? |
| **Versioning** | Semantic versioning? Git tags? Changelog automation? |

### 2.5 What NOT to Do (Anti-patterns for MVP)

List specific anti-patterns to avoid:

- Heavy full-stack starters (T3, RedwoodJS, etc.)
- Premature microservices or message queues
- Coupling CalDAV state to React stores (source of truth confusion)
- Over-engineering auth before core CalDAV works
- Complex build pipelines before basic functionality
- Feature flags / A/B testing infrastructure
- Multi-tenant / multi-user patterns in MVP
- (Add more based on research)

---

## 3. Constraints / Non-goals

**Hard constraints:**
- No heavy full-stack starter — keep `web/` and `server/` decoupled
- CalDAV subset minimal — sync correctness > feature breadth
- Design system framework-agnostic — don't force Radix-only language; mention alternatives
- No secrets or credentials in output — docs-only, no vendor keys
- SQLite for MVP — but schema must support future Postgres migration

**Non-goals (out of scope for this research):**
- Actual implementation code
- LLM provider selection or prompt engineering
- Todoist API integration details
- CI/CD vendor selection (GitHub Actions assumed)
- Production infrastructure (reverse proxy, TLS, etc.)

---

## 4. Deliverables

GPT-5 Pro must output the following sections:

### 4.1 Decision Matrix

For each area (Go backend, React frontend, monorepo, deployment), provide:

| Option | Pros | Cons | Recommended? |
|--------|------|------|--------------|
| Option A | ... | ... | Yes/No |
| Option B | ... | ... | Yes/No |
| Option C | ... | ... | Yes/No |

### 4.2 Recommended Default Path

A single coherent recommendation with rationale:
- Why these choices fit Calendar-app's requirements
- How they support graceful degradation
- How they enable the Propose → Confirm → Apply model
- Trade-offs accepted and why

### 4.3 Proposed Repository Tree

Concrete directory structure:

```
Calendar-app/
├── server/
│   ├── cmd/
│   ├── internal/
│   ├── ...
├── web/
│   ├── src/
│   ├── ...
├── docs/
├── scripts/
├── docker/
├── ...
```

With brief explanation of each directory's purpose.

### 4.4 Bootstrap Steps

Concrete commands and minimal files to initialize the repo:

```bash
# Example format
mkdir -p server/cmd/calendar-app
go mod init github.com/user/calendar-app/server
# ...
```

Include:
- Go module initialization
- React/Vite project creation
- Tailwind setup
- Dev script for running both
- Basic Makefile or Taskfile

### 4.5 Risks & Validation Steps

| Risk | Mitigation | Validation Step |
|------|------------|-----------------|
| CalDAV client compatibility | ... | Interop test with Fantastical, Apple Calendar, DAVx⁵ |
| SQLite concurrent access | ... | Load test with multiple CalDAV clients |
| ... | ... | ... |

Include a suggested **interop test plan** for CalDAV clients.

---

## 5. Acceptance Criteria

The research output is acceptable if:

- [ ] **Actionable:** Can start implementation without further research
- [ ] **Scoped:** Clearly limited to MVP; avoids over-engineering
- [ ] **Traced:** Recommendations explicitly tie back to PRD/UX requirements
- [ ] **Concrete:** Includes actual commands, file paths, and code snippets where helpful
- [ ] **Balanced:** Presents trade-offs honestly; doesn't oversell any option
- [ ] **Validated:** Includes verification steps for risky decisions (especially CalDAV interop)

---

## 6. Reference Documents

For full context, review these documents in the Calendar-app repository:

| Document | Path | Key Content |
|----------|------|-------------|
| PRD | `_bmad-output/planning-artifacts/prd.md` | 65 FRs, 6 NFR categories, user journeys |
| PRD Validation | `_bmad-output/planning-artifacts/prd-validation-report.md` | PASS status, quality assessment |
| UX Spec | `_bmad-output/planning-artifacts/ux-design-specification.md` | Design system, interaction patterns |
| Architecture (WIP) | `_bmad-output/planning-artifacts/architecture.md` | Project context analysis |

---

## 7. Output Format

Return your research as a single markdown document with clear H2 sections matching the deliverables above. Use tables for decision matrices. Include code blocks for commands and file structures.

**Do not include:**
- Actual implementation code beyond minimal bootstrapping
- Secrets, API keys, or credentials
- Time estimates
- Marketing language or hype

**Do include:**
- Specific version numbers where relevant (e.g., "Go 1.22+", "Vite 5.x")
- Links to official documentation
- Explicit reasoning for each recommendation
