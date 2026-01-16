# internal/jobs

Background job scheduling and execution.

## Responsibilities

- Todoist sync scheduler (periodic)
- AI plan generation triggers (time-based)
- Calendar change monitoring
- Health checks for external services

## Key Files (to be created)

- `scheduler.go` - Job scheduling setup (using robfig/cron or go-co-op/gocron)
- `todoist_sync.go` - Todoist sync job implementation
- `plan_generator.go` - AI plan generation job
