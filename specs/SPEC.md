# Update Plugin Specification

## Purpose

OS and package update management for headless Debian-based nodes. This plugin
provides a safe, remotely-controllable way to list, apply, and schedule system
updates without interactive access.

## Responsibilities

- **List pending updates** — query `apt` for available package upgrades and
  report counts, package names, and severity levels.
- **Run security-only updates** — apply only updates from the configured
  security pocket (`-security`), minimizing risk.
- **Run full upgrades** — apply all pending upgrades (`apt-get dist-upgrade`).
- **View last run status/logs** — persist the outcome (success/failure, start
  time, duration, packages affected) so it can be queried later.
- **Schedule automatic updates** — expose a cron-driven job that the core
  scheduler can trigger on a configurable cadence.

## Integration

- Implements the local `pluginiface.Plugin` interface used by this repository.
- Does **not** call `plugin.Register()` in `init()`; registration is performed
  explicitly by the core integration layer when constructing the plugin.
- Included in the core binary via the normal dependency graph; the core wiring
  code instantiates and registers the plugin.
- Routes are mounted by the core API server under
  `/api/v1/plugins/update`.

## API Routes

All routes are relative to the plugin mount point (`/api/v1/plugins/update`).

| Method | Path      | Description                                   |
|--------|-----------|-----------------------------------------------|
| GET    | `/status` | List pending updates and current system state  |
| POST   | `/run`    | Trigger an update run (`type`: security\|full) |
| GET    | `/logs`   | Retrieve logs from the last update run         |
| GET    | `/config` | Return current plugin configuration            |

### POST /run Request Body

```json
{
  "type": "security"
}
```

`type` must be either `"security"` or `"full"`.

### Error Format

Errors follow the core convention:

```json
{
  "error": {
    "code": 400,
    "message": "invalid update type",
    "details": "type must be 'security' or 'full'"
  }
}
```
