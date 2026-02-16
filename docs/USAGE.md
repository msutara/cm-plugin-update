# Usage

## Overview

The update plugin is integrated into Config Manager by importing it and
registering the plugin with the core's plugin registry. Once integrated,
all endpoints are available under `/api/v1/plugins/update`.

> **Note:** In Phase 1, the plugin uses a local `pluginiface` package that
> mirrors the core's `plugin.Plugin` interface for independent development.
> Full integration with the core binary will be wired in Phase 2.

## Endpoints

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

## Scheduled Jobs

| Job ID            | Schedule  | Description                      |
|-------------------|-----------|----------------------------------|
| update.security   | 0 3 * * * | Run automatic security updates   |
