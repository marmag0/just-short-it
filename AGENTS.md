# AGENTS.md

## Project overview

- Project brief: a high-efficiency link shortener monorepo with no frontend, focused on telemetry and data gathering. Links are managed from a CLI over an HTTP API. Short links use custom user-defined suffixes (e.g. `/{custom_suffix}`) and temporary HTTP redirects (`307`/`302`) to ensure all clicks trigger telemetry collection. Telemetry is passive (request headers, Cloudflare geolocation headers, and Client Hints, no JavaScript) and stored in ClickHouse. `[complete here: final telemetry data points and ClickHouse schema]`
- Privacy: client IP addresses are never stored or logged raw. `shortener` replaces them with a keyed hash (HMAC-SHA256) before a click event leaves the process.
- Planned stack: Go for the backend services and the CLI, PostgreSQL for links and API keys, Redis for URL caching and buffering click events (Redis Streams), ClickHouse for click analytics, Grafana for visualization, Loki and Tempo for service logs and traces collected by Grafana Alloy, dbmate for database migrations, Cloudflare with a `cloudflared` tunnel in front, and Docker with Docker Compose for orchestration on a single VPS.
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
- **Go** (`services/`, `cli/`, `tests/`): Format with `gofmt` and lint with `go vet`. Prefer the standard library and do not add linters or dependencies without a clear need.
- **Whitespace and line breaks**:
  - Use blank lines purposefully to separate logical units, not randomly.
  - SQL migrations (`services/db/*/migrations/`):
    - File naming follows a two-digit prefix and camelCase name (e.g. `01_createUrlsTable.sql`, `02_addTelemetryEvents.sql`).
    - Add a blank line immediately after `-- migrate:up` and `-- migrate:down`.
    - Group distinct logical blocks with a single blank line (e.g. between `DO $$ ... $$;` blocks, `GRANT` statements, and `ALTER DEFAULT PRIVILEGES`).
    - Separate the end of the `migrate:up` section from the `-- migrate:down` header with two blank lines.
  - Shell scripts (`services/db/postgres/init/`):
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
  - `services/<service>/cmd/`: `main` package that configures and starts the service.
  - `services/<service>/<package>/`: Go packages with the service logic (e.g. `services/shortener/link/`).
  - `services/<service>/deploy/`: Service containerization assets (`Dockerfile`) and deployment manifests.
- Initial services:
  - `services/shortener/`: Go service handling high-performance temporary redirection (`307`/`302`) and the link management API, on separate listeners. Checks Redis cache before querying PostgreSQL. Pushes click events to a Redis Stream without blocking the redirect.
  - `services/telemetry/`: Go worker that reads click events from the Redis Stream, enriches them (User-Agent parsing, bot detection), and batch-inserts them into ClickHouse.
  - Extension point: Future microservices will live under `services/<service_name>/` following the same `cmd/` and `deploy/` convention.
- `cli/`: Go link management CLI (create, list, edit, and disable links, generate QR codes). Talks to the management API with a per-person API key.
- `services/db/`: Centralized database directory. It is not a Go service and does not follow the `cmd/` and `deploy/` convention:
  - `services/db/postgres/init/`: Container bootstrap scripts (e.g. application user creation).
  - `services/db/postgres/migrations/`: `dbmate` SQL migrations for links and API keys.
  - `services/db/clickhouse/migrations/`: `dbmate` SQL migrations for click events.
  - Schema snapshot generation is disabled (no `schema.sql`).
- `services/grafana/`, `services/loki/`, `services/tempo/`, `services/alloy/`: Observability stack configuration, mounted read-only into the containers. `services/grafana/provisioning/` holds the ClickHouse, Loki, and Tempo datasources and dashboards. Like `services/db/`, these are not Go services and do not follow the `cmd/` and `deploy/` convention.
- `tests/`: All automated tests, grouped by service:
  - `tests/<service>/<package>/`: Unit tests of `services/<service>/<package>/` in the external test package `<package>_test`.
  - `tests/<service>/e2e/`: End-to-end tests that build the service binary, run it, and send it HTTP requests.
  - `tests/stack/`: Tests of `docker-compose.yml` (ports come from `.env`), the Loki, Tempo, and Alloy configurations, the `shortener` image, and the whole running stack. They need Docker and are skipped without it.
  - Tests reach only exported names of non-`main` packages, so testable logic lives outside `cmd/`.
- `.agents/skills/`: Agent skills shared by Codex and Claude Code, one directory per skill (`<skill>/SKILL.md`):
  - Codex reads `.agents/skills/` directly. This is the only place to edit skills.
  - Claude Code reads `.claude/skills/`, where each `<skill>` is a relative symlink to `.agents/skills/<skill>/`. Add a matching symlink for every new skill.
  - On Windows, the symlinks require `core.symlinks=true` in Git.
- `docker-compose.yml`: Top-level multi-service orchestration defining:
  - `cloudflared`: Cloudflare tunnel, the only way in from the internet (other host ports bind to loopback only). Runs only with the `tunnel` profile.
  - `db`: PostgreSQL database.
  - `db-migrator`: `dbmate` one-shot migration runner for PostgreSQL.
  - `clickhouse`: ClickHouse database for click events.
  - `clickhouse-migrator`: `dbmate` one-shot migration runner for ClickHouse.
  - `redis`: In-memory cache for fast URL redirection and buffer for click events.
  - `shortener`: Link shortener and redirect service (`services/shortener/deploy/Dockerfile`).
  - `telemetry`: Click event ingestion worker (`services/telemetry/deploy/Dockerfile`).
  - `grafana`: Metrics and telemetry visualization dashboards.
  - `loki`: Log storage for container logs.
  - `tempo`: Trace storage for OpenTelemetry traces.
  - `alloy`: Collector that ships container logs to Loki and OTLP traces to Tempo.
  - Extension point: Additional telemetry sinks or services.
- `go.mod`: The only Go module (`github.com/marmag0/just-short-it`), shared by all services, the CLI, and the tests.
- `Makefile`: Monorepo task shorthand (`make dev`, `make prod`, `make down`, `make test`, `make lint`, `make format`, `make help`).
- `.env`: Local environment values (ignored by Git). `.env.example`: Tracked template with variable placeholders.
- `README.md`: Top-level project documentation.

## Validation

- Run `git diff --check` after changes and inspect changed and newly created files for unintended edits.
- For documentation changes, check grammar, Markdown structure, and affected links or table-of-contents anchors.
- For ignore-rule changes, check representative ignored and tracked paths with `git check-ignore`.
- For Go changes, run `gofmt -l .`, `go vet ./...`, and `go test ./...` from the repository root. `gofmt -l` must print nothing. Apply fixes with `gofmt -w .`. `go test -short ./...` skips the slow whole-stack test.
- To measure coverage of `services/`, including the e2e binaries built with `-cover`, run `go test -short -cover -coverpkg=./services/... ./tests/... -args -test.gocoverdir=<dir>`, then `GOCOVERDIR=<dir> go test -count=1 ./tests/shortener/e2e/`, and read the total with `go tool covdata percent -i=<dir>`. `<dir>` must be an existing absolute path outside the repository, because tests run in other working directories.
- For shell script changes in `services/db/postgres/init/`, run `sh -n services/db/postgres/init/*.sh`.
- For code or Docker changes, run the relevant configured formatting, build, test, and configuration checks across affected services. Add or update regression tests in `tests/` when behavior changes warrant them.

## Secrets and local files

- Never commit real credentials, tokens, private keys, or populated secret files. Use placeholders in tracked examples and documentation.
- Keep secrets out of logs, command output, and shared VS Code configuration. `.gitignore` is not a substitute for reviewing files before staging.
- Store local secret values in `.env`. Keep `.env.example` tracked with the required variable names and placeholder values only, never real secrets.
