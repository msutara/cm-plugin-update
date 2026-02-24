# cm-plugin-update

OS and package update management plugin for
[Config Manager](https://github.com/msutara/config-manager-core). Designed for
headless Debian-based nodes (Raspbian Bookworm ARM64, Debian Bullseye slim).

## Features

- List pending OS and package updates with severity classification
- Run security-only updates to minimize risk
- Run full system upgrades (`apt-get dist-upgrade`)
- View last run status, duration, and logs
- Schedule automatic security updates via the core scheduler
- RESTful API mounted at `/api/v1/plugins/update`

## Documentation

- [Usage Guide](docs/USAGE.md) — endpoint examples and scheduled jobs
- [Specification](specs/SPEC.md) — responsibilities, integration, API routes

## Development

```bash
# lint
golangci-lint run

# test
go test ./...
```

CI runs automatically on push/PR to `main` via GitHub Actions
(`.github/workflows/ci.yml`).

## License

License not yet finalized.
