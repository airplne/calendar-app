# Calendar-app

AI planning co-pilot built on a self-hosted CalDAV server. Delivers ambient protection over ritual optimization through constraint-guaranteed AI planning.

## Overview

Calendar-app is a model-agnostic planning co-pilot that:

- Runs a first-class **CalDAV server** as the calendar source of truth (sync with Fantastical/Apple Calendar/Android CalDAV clients)
- Integrates with **Todoist** (read/write) for tasks and projects
- Uses an **AI harness** (Grok/Claude/Gemini/OpenAI/etc.) to propose plans with strict `propose → confirm → apply` workflow
- Provides **ambient protection:** continuous monitoring, interruption detection, and adaptive planning (no ritual planning sessions)
- Ensures **constraint-guaranteed AI:** deterministic validation gates all proposals before display or apply

## Architecture

- **Backend:** Go 1.22+ (CalDAV/WebDAV + REST API)
- **Frontend:** React 18+ SPA via Vite 5.x
- **Database:** SQLite with WAL mode (schema designed for Postgres migration)
- **Deployment:** Single binary (Go embeds React dist) or Docker

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 18+ and pnpm
- Make (optional, but recommended)

### Development

```bash
# Install dependencies
make install-deps

# Run both server and web in dev mode
make dev
```

- Go server: http://localhost:8080
- React dev server: http://localhost:5173 (proxies API to :8080)

### Build

```bash
# Build production binary
make build

# Run the binary
./calendar-app
```

### Project Structure

```
Calendar-app/
├── server/              # Go backend
│   ├── cmd/
│   │   └── calendarapp/ # Main entry point
│   ├── internal/        # Application code
│   │   ├── caldav/      # CalDAV/WebDAV handlers
│   │   ├── api/         # REST API handlers
│   │   ├── jobs/        # Background scheduler
│   │   ├── integrations/# Todoist, LLM clients
│   │   └── data/        # DB models, migrations
│   └── migrations/      # SQL migrations (goose)
├── web/                 # React frontend
│   ├── src/
│   │   ├── components/  # UI components
│   │   ├── pages/       # Route pages
│   │   └── lib/         # Utils, API client
│   ├── index.html
│   └── package.json
├── scripts/             # Dev/build scripts
├── docker/              # Dockerfile
└── docs/                # Documentation
```

## Configuration

Environment variables (all optional, with defaults):

- `CALENDARAPP_PORT` - HTTP port (default: `8080`)
- `CALENDARAPP_DATA_DIR` - Data directory for SQLite (default: `./data`)
- `CALENDARAPP_TODOIST_TOKEN` - Todoist API token
- `CALENDARAPP_LLM_API_KEY` - LLM provider API key

Alternatively, create `config.yaml` in the working directory.

## Documentation

Project planning artifacts live in `_bmad-output/`:

- [Architecture Decision Document](_bmad-output/planning-artifacts/architecture.md)
- [Product Requirements Document](_bmad-output/planning-artifacts/prd.md)
- [UX Design Specification](_bmad-output/planning-artifacts/ux-design-specification.md)

## License

TBD
