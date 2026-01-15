# calendar-app

Calendar-app is a model-agnostic, chat-first planning co-pilot that:

- Runs a first-class **CalDAV server** as the calendar source of truth (sync with Fantastical/Apple Calendar/Android CalDAV clients).
- Integrates with **Todoist** (read/write) for tasks and projects.
- Uses an **AI harness** (Grok/Claude/Gemini/OpenAI/etc.) to propose plans, with a strict `propose → confirm → apply` workflow and an audit/undo log.

Project planning artifacts live in `_bmad-output/`.
