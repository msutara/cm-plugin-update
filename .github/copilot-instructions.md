# Copilot Instructions

## Project Overview

cm-plugin-update is a Go plugin for Config Manager that handles OS and package
update management on headless Debian-based nodes. It provides endpoints to list
pending updates, trigger security-only or full upgrades, view run logs, and
schedule automatic updates via the core scheduler.

Target platforms: Raspbian Bookworm (ARM64), Debian Bullseye slim.

## Architecture

- **plugin.go** — `UpdatePlugin` struct implementing `plugin.Plugin` from `config-manager-core`;
  registration handled by the core (no `init()` self-registration)
- **routes.go** — Chi router with handlers for `/status`, `/run`, `/logs`,
  `/config`; mounted by the core under `/api/v1/plugins/update`
- **service.go** — domain logic: `ListPendingUpdates`, `RunSecurityUpdates`,
  `RunFullUpgrade`, `GetLastRunStatus`

## Integration

The plugin is compiled into the core binary via a normal import in
`cmd/cm/main.go`:

```go
import update "github.com/msutara/cm-plugin-update"

plugin.Register(update.NewUpdatePlugin())
```

Routes are mounted under `/api/v1/plugins/update`.

## Conventions

- Main Go package is `package update` at the repo root
- Additional helper packages are allowed
- Use `github.com/go-chi/chi/v5` for HTTP routing
- Use `log/slog` for all structured logging (include `"plugin", "update"`)
- Error responses: `{"error": {"code": ..., "message": ..., "details": ...}}`
- Job IDs follow the pattern `update.{job_name}`
- Specs live in `specs/`, user docs in `docs/`
- Filenames use UPPERCASE-KEBAB-CASE (e.g., `SPEC.md`, `USAGE.md`)

## Specifications

- [specs/SPEC.md](../specs/SPEC.md) — plugin specification and scope
- [docs/USAGE.md](../docs/USAGE.md) — endpoint examples and scheduled jobs

## Validation

- All Go code must pass `golangci-lint run`
- All tests must pass: `go test ./...`
- CI runs markdownlint + lint + test via `.github/workflows/ci.yml`
- Never push directly to main — always use feature branches and PRs
