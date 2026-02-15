# Usage

## Overview

The update plugin is loaded automatically when the Config Manager binary
starts. All endpoints are available under `/api/v1/plugins/update`.

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
