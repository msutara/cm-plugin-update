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

> **Note:** In Phase 1, the plugin uses a local `pluginiface` package that
> mirrors the core's `plugin.Plugin` interface for independent development.

## 3. API Endpoints

### Check pending updates

```bash
curl http://localhost:8080/api/v1/plugins/update/status
```

### Run security-only updates

```bash
curl -X POST http://localhost:8080/api/v1/plugins/update/run \
  -H "Content-Type: application/json" \
  -d '{"type": "security"}'
```

### Run full upgrade

```bash
curl -X POST http://localhost:8080/api/v1/plugins/update/run \
  -H "Content-Type: application/json" \
  -d '{"type": "full"}'
```

### View last run logs

```bash
curl http://localhost:8080/api/v1/plugins/update/logs
```

### View plugin configuration

```bash
curl http://localhost:8080/api/v1/plugins/update/config
```

## 4. Scheduled Jobs

| Job ID            | Schedule      | Description                    |
|-------------------|---------------|--------------------------------|
| update.security   | `0 3 * * *`  | Run automatic security updates |

## 5. Configuration

The plugin exposes a read-only configuration view via `GET /config`:

```json
{
  "auto_security_updates": true,
  "schedule": "0 3 * * *"
}
```

These are plugin-defined, read-only defaults; they are not directly
user-configurable.
