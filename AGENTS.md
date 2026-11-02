# AGENTS.md

## Project overview

- Project brief: a high-efficiency link shortener monorepo with no frontend, focused on telemetry and data gathering. Short links use custom user-defined suffixes (e.g. `/{custom_suffix}`) and temporary HTTP redirects (`307`/`302`) to ensure all clicks trigger telemetry collection. `[complete here: telemetry data points, metrics schema, and storage destination when chosen]`
- Planned stack: Python with FastAPI for the backend services, PostgreSQL for persistent data, Redis for in-memory URL caching, Grafana for metrics visualization, dbmate for database migrations, and Docker with Docker Compose for orchestration.
- Development environment: VS Code on macOS, Windows, and Linux. `.gitignore` should cover generated artifacts from the stack and development environments, as well as local secret files.

## Working rules

- **Simplicity is the key**: Keep things simple. Do not generate boilerplate, overengineered abstractions, or premature complexity. Write minimal, clean, and direct code that solves the task directly.
- Always use classic dashes (`-`) rather than em dashes in code, comments, documentation, commit messages, and responses.
- Follow the repository's existing code structure, formatting, commenting, and documentation style. Read nearby files before editing and use established tooling when available.
- Never delete comments written by the user. If a comment conflicts with a requested change, point out the conflict and ask the user how to resolve it. The only exception is a TODO comment addressed by your changes, which may be removed when replaced with the corresponding implementation.
- Add code comments only when needed. Keep them simple, straightforward, and idiomatic, without excessive formatting. Avoid decorative, obvious, or boilerplate comments.
- Make only changes needed to complete the requested task. Report unrelated errors or improvement opportunities instead of fixing them without being asked.
- Inspect the working tree before editing and preserve unrelated or pre-existing changes.
- Challenge unclear requirements or assumptions. Ask the user for clarification before implementing an ambiguous decision rather than inventing behavior, architecture, or scope. Continue independent work that does not depend on the answer.
- Validate that changes are correct and do not break related behavior. Review the final diff, run relevant available checks, and report their results and any checks that could not be run. Use `make lint` and `make test` for validation.
- Use `.gitkeep` to include otherwise empty directories in Git. Remove a directory's `.gitkeep` once it contains other tracked files.

## Code style and formatting conventions

- **Core philosophy**: Simplicity and readability. Every line, comment, and configuration entry must have a clear purpose. Avoid boilerplate, speculative features, or redundant helpers.
- **Dashes**: Always use classic ASCII dashes (`-`) rather than em dashes or en dashes.
- **Python** (`services/*/app/`): Format and lint with Ruff using its default configuration. Do not add custom Ruff rules without a clear need.
- **Whitespace and line breaks**:
  - Use blank lines purposefully to separate logical units, not randomly.
  - SQL migrations (`db/migrations/`):
    - File naming follows a two-digit prefix and camelCase name (e.g. `01_createUrlsTable.sql`, `02_addTelemetryEvents.sql`).
    - Add a blank line immediately after `-- migrate:up` and `-- migrate:down`.
    - Group distinct logical blocks with a single blank line (e.g. between `DO $$ ... $$;` blocks, `GRANT` statements, and `ALTER DEFAULT PRIVILEGES`).
    - Separate the end of the `migrate:up` section from the `-- migrate:down` header with two blank lines.
  - Shell scripts (`db/init/`):
    - Use POSIX shell (`#!/bin/sh`) with `set -e`.
    - Place a single blank line after the shebang and after `set -e`.
    - Keep scripts concise, minimal, and idiomatic without wrapper bloat.
  - Environment files (`.env`, `.env.example`):
    - Logically group related variables (ports, database credentials, Redis configuration, Grafana settings, service secrets).
    - Separate variable groups with a single blank line.
    - Avoid noisy inline comments; let variable names remain self-explanatory.
  - Docker Compose (`docker-compose.yml`):
    - Use strict 2-space indentation for hierarchy.
    - Separate service definitions with a single blank line.
    - Format multi-line array arguments with clean line breaks (such as healthcheck `test`).
    - Use the `${VAR:?Set VAR in .env}` pattern for required environment variables.
    - Preserve section comments that indicate extension points (e.g. `# other services can go here`).
- **Comments**:
  - Add comments only when the intent is not self-evident from the code or when marking extension points.
  - Never write decorative, redundant, or boilerplate comments that merely restate what the code does.

## Architecture and repository layout

- `services/` contains application microservices. Each service is structured as:
  - `services/<service>/app/`: Application source code, configuration, dependency management (managed with uv), and automated tests.
  - `services/<service>/deploy/`: Service containerization assets (`Dockerfile`) and deployment manifests.
- Initial services:
  - `services/shortener/`: Python FastAPI service handling link management and high-performance temporary redirection (`307`/`302`). Checks Redis cache before querying PostgreSQL.
  - Extension point: Future microservices (such as a dedicated telemetry ingestion worker or stream processor) will live under `services/<service_name>/` following the same `app/` and `deploy/` convention.
- `db/`: Centralized PostgreSQL database directory at repository root:
  - `db/init/`: Container bootstrap scripts (e.g. application user creation).
  - `db/migrations/`: `dbmate` SQL migrations. Schema snapshot generation is disabled (no `db/schema.sql`).
- `docker-compose.yml`: Top-level multi-service orchestration defining:
  - `db`: PostgreSQL database.
  - `db-migrator`: `dbmate` one-shot migration runner.
  - `redis`: In-memory cache for fast URL redirection.
  - `shortener`: FastAPI link shortener and redirect service (`services/shortener/deploy/Dockerfile`).
  - `grafana`: Metrics and telemetry visualization dashboards.
  - Extension point: Additional telemetry sinks or services.
- `Makefile`: Monorepo task shorthand (`make dev`, `make prod`, `make down`, `make test`, `make lint`, `make format`, `make help`).
- `.env`: Local environment values (ignored by Git). `.env.example`: Tracked template with variable placeholders.
- `README.md`: Top-level project documentation.

## Validation

- Run `git diff --check` after changes and inspect changed and newly created files for unintended edits.
- For documentation changes, check grammar, Markdown structure, and affected links or table-of-contents anchors.
- For ignore-rule changes, check representative ignored and tracked paths with `git check-ignore`.
- For Python changes in any service, run `uv run --directory services/<service>/app ruff check`, `uv run --directory services/<service>/app ruff format --check`, and `uv run --directory services/<service>/app pytest`. Apply fixes with `ruff check --fix` and `ruff format`.
- For shell script changes in `db/init/`, run `sh -n db/init/*.sh`.
- For code or Docker changes, run the relevant configured formatting, build, test, and configuration checks across affected services. Add or update regression tests when behavior changes warrant them.

## Secrets and local files

- Never commit real credentials, tokens, private keys, or populated secret files. Use placeholders in tracked examples and documentation.
- Keep secrets out of logs, command output, and shared VS Code configuration. `.gitignore` is not a substitute for reviewing files before staging.
- Store local secret values in `.env`. Keep `.env.example` tracked with the required variable names and placeholder values only, never real secrets.
