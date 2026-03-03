# Update Plugin Specification

## 1. Purpose

OS and package update management for headless Debian-based nodes. This plugin
provides a safe, remotely-controllable way to list, apply, and schedule system
updates without interactive access.

## 2. Responsibilities

- **List pending updates** — query `apt` for available package upgrades and
  report counts, package names, and severity levels.
- **Run security-only updates** — apply only updates from the configured
  security pocket (`-security`), minimizing risk.
- **Run full upgrades** — apply all pending upgrades (`apt-get dist-upgrade`).
- **View last run status/logs** — persist the outcome (success/failure, start
  time, duration, packages affected) so it can be queried later.
- **Schedule automatic updates** — expose a cron-driven job (`0 3 * * *` by
  default) that the core scheduler triggers automatically.

## 3. Non-responsibilities

- No `apt-key` management or GPG key handling
- No PPA or third-party repository management
- No rollback of applied updates
- No kernel live-patching or reboot orchestration

## 4. Integration

- Implements the core `plugin.Plugin` interface from `config-manager-core`.
- Does **not** call `plugin.Register()` in `init()`; registration is performed
  explicitly by the core integration layer when constructing the plugin.
- Included in the core binary via the normal dependency graph; the core wiring
  code instantiates and registers the plugin.
- Routes are mounted by the core API server under
  `/api/v1/plugins/update`.
- Scheduled jobs are registered with the core scheduler at startup.

## 5. API Routes

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

Only one update can run at a time. If a run is already in progress, the
endpoint returns `409 Conflict`:

```json
{
  "error": {
    "code": 409,
    "message": "update already running",
    "details": "an update is already running"
  }
}
```

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

## 6. Scheduled Jobs

| Job ID            | Default Schedule | Description                    |
|-------------------|------------------|--------------------------------|
| update.full       | *(none)*         | Run full system upgrade        |
| update.security   | `0 3 * * *`     | Run security updates           |

> `update.full` is always registered with no cron schedule (manual trigger
> only via the jobs API).
>
> `update.security` is registered when the security source is available:
> when `security_source` is `"detected"`, the system must have a separate
> security apt source (e.g. Debian); when set to `"always"`, the job is
> registered regardless. The cron schedule is attached only when
> `auto_security` is enabled; otherwise the job is manual-trigger-only.

## 7. Configuration

The plugin exposes a read-only configuration view via `GET /config`:

```json
{
  "schedule": "0 3 * * *",
  "auto_security": true,
  "security_source": "detected",
  "security_available": true
}
```

| Field                | Type   | Description                                      |
| -------------------- | ------ | ------------------------------------------------ |
| `schedule`           | string | Cron expression for automatic security updates   |
| `auto_security`      | bool   | Whether automatic security updates are enabled   |
| `security_source`    | string | `"detected"` or `"always"` — controls gating    |
| `security_available` | bool   | Read-only; computed once at startup, not persisted in config |

`security_available` is determined once during service initialization by
probing the system's apt sources. The cached value is returned in every
`/config` response for informational purposes but is not a configurable
setting.

When `security_source` is `"detected"` and the system lacks a separate
security apt source, the scheduled job is omitted. When set to `"always"`,
the job runs regardless.
Configuration is managed via the core's settings endpoint
(`PUT /api/v1/plugins/update/settings`).

## 8. Concurrency

- **Config access** is protected by a `sync.RWMutex`; concurrent reads via
  `CurrentConfig()` and `ScheduledJobs()` do not block each other, while
  writes via `Configure()` and `UpdateConfig()` are serialized.
- **Init** uses `sync.Once` so the startup probe runs exactly once, even if
  `Init()` is called from multiple goroutines.
- **Update runs** are guarded by a running flag: if a second `/run` request
  arrives while an update is executing, it is rejected immediately with
  `409 Conflict` rather than queuing.
- **GetLastRunStatus** returns a defensive deep copy so callers cannot
  mutate internal state.

## 9. Resource Limits

- **Log output** stored in `RunStatus.Log` is capped at 64 KB total,
  including any truncation marker. When apt output exceeds the limit,
  `RunStatus.Log` consists of a `...(truncated)\n` marker followed by the
  last bytes of apt output such that the total length never exceeds 64 KB.
  The full apt output is still parsed for package counts before truncation.
