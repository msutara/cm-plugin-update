module github.com/msutara/cm-plugin-update

// Note: The spec references github.com/msutara/config-manager-core/internal/plugin.
// That dependency is intentionally omitted so this plugin can be developed and
// tested independently. The local pluginiface package mirrors the core Plugin
// interface; verify compatibility before integrating with config-manager-core.

go 1.22

require github.com/go-chi/chi/v5 v5.1.0
