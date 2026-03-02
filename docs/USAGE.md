# Usage

## 1. Overview

The update plugin manages OS and package updates on headless Debian-based
nodes. It exposes endpoints to list pending updates, trigger security-only
or full upgrades, view run logs, and retrieve configuration. All endpoints
are available under `/api/v1/plugins/update`.

## 2. Integration

The plugin is integrated into Config Manager by importing it and registering
it with the core's plugin registry:

```go
import update "github.com/msutara/cm-plugin-update"

plugin.Register(update.NewUpdatePlugin())
```

> **Note:** The plugin implements the `plugin.Plugin` interface from
> `config-manager-core` directly.

## 3. API Endpoints

### Check pending updates

```bash
curl http://localhost:7788/api/v1/plugins/update/status
```

### Run security-only updates

```bash
curl -X POST http://localhost:7788/api/v1/plugins/update/run \
  -H "Content-Type: application/json" \
  -d '{"type": "security"}'
```

### Run full upgrade

```bash
curl -X POST http://localhost:7788/api/v1/plugins/update/run \
  -H "Content-Type: application/json" \
  -d '{"type": "full"}'
```

### View last run logs

```bash
curl http://localhost:7788/api/v1/plugins/update/logs
```

### View plugin configuration

```bash
curl http://localhost:7788/api/v1/plugins/update/config
```

## 4. Scheduled Jobs

| Job ID            | Default Schedule | Description                    |
|-------------------|------------------|--------------------------------|
| update.security   | `0 3 * * *`     | Run automatic security updates |

## 5. Configuration

The plugin exposes a read-only configuration view via `GET /config`:

```json
{
  "schedule": "0 3 * * *",
  "auto_security": true,
  "security_source": "available",
  "security_available": true
}
```

| Field                | Type   | Description                                      |
| -------------------- | ------ | ------------------------------------------------ |
| `schedule`           | string | Cron expression for automatic security updates   |
| `auto_security`      | bool   | Whether automatic security updates are enabled   |
| `security_source`    | string | `"available"` or `"always"` — controls gating    |
| `security_available` | bool   | Read-only; computed once at startup, not persisted in config |

`security_available` is determined once during service initialization by
probing the system's apt sources. The cached value is returned in every
`/config` response for informational purposes but is not a configurable
setting.

When `security_source` is `"available"` and the system lacks a separate
security apt source, the scheduled job is omitted. When set to `"always"`,
the job runs regardless of source availability.
