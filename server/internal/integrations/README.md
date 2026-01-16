# internal/integrations

External service clients (Todoist, LLM providers).

## Responsibilities

- Todoist API client with OAuth/token handling
- LLM provider client (model-agnostic harness)
- Graceful degradation on service failures
- Interface definitions for mocking in tests

## Key Files (to be created)

- `todoist/client.go` - Todoist API client
- `llm/client.go` - LLM provider interface and implementations
- `llm/openai.go` - OpenAI-specific implementation
- `interfaces.go` - Common interfaces for external services
