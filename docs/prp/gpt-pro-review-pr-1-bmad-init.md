# PRP: Pull Request Packet

PR: https://github.com/airplne/calendar-app/pull/1

Status: merged to `main` via squash (merge commit `f320e2e`, 2026-01-16); branch `bmad-initial-artifacts` deleted.

## 1. Context / Goal

Calendar-app is a model-agnostic, chat-first calendar + task assistant with a CalDAV server and Todoist integration. This PR bootstraps the BMAD framework scaffold and captures initial planning artifacts from the brainstorming phase.

## 2. What’s Included

- `_bmad/` — BMAD workflow framework scaffold (507 files; treat as vendor-like framework code)
- `_bmad-output/analysis/brainstorming-session-2026-01-15.md` — Initial product brainstorming session
- `_bmad-output/planning-artifacts/prd.md` — PRD in progress (classification + success criteria + scope captured; remaining sections to be completed via BMAD PRD workflow)
- `_bmad-output/planning-artifacts/bmm-workflow-status.yaml` — BMAD workflow state tracker
- `.gitignore`, `README.md`, `CLAUDE.md`, `.mcp.json`

## 3. What’s Not Included

- No implementation or application code
- PRD is partial; next PR will complete remaining PRD sections (journeys, domain, FRs, NFRs, polish/validation) and proceed to architecture/epics

## 4. Key Decisions Captured

- CalDAV server as calendar source of truth (Fantastical/Apple Calendar/Android clients sync to us)
- Todoist REST API for task management (read/write)
- Model-agnostic AI harness (Grok, Claude, Gemini, OpenAI, etc.)
- Propose → confirm → apply pattern for AI-initiated writes
- Audit log with undo capability

## 5. Review Focus for GPT Pro

- Confirm no secrets or credentials committed
- Confirm `.gitignore` excludes `.codex/`, `.claude/`, `.agentvibes/`, `.env*`
- Confirm CalDAV decision is documented in `_bmad-output/analysis/brainstorming-session-2026-01-15.md`
- Sanity-check artifacts are organized and internally consistent

## 6. How to Validate

- Docs-only PR; no build/tests expected
- Verify file presence: `ls _bmad/ _bmad-output/`
- Verify no sensitive dirs tracked: `git ls-files | grep -E '^\\.(codex|claude|agentvibes)/'` should return empty

## 7. Risks / Follow-ups

- Large vendor-like addition of BMAD framework files (507 files); do not deep-review individual scaffold files
- Next PR will complete remaining PRD sections and proceed to architecture/epics
