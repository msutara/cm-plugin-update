package update

import (
	"fmt"
	"net/http"

	"github.com/msutara/config-manager-core/plugin"
)

// Compile-time checks.
var (
	_ plugin.Plugin       = (*UpdatePlugin)(nil)
	_ plugin.Configurable = (*UpdatePlugin)(nil)
)

// Default configuration values.
const (
	DefaultSchedule       = "0 3 * * *"
	DefaultAutoSecurity   = true
	DefaultSecuritySource = "available"
)

// Note: Registration with the core is handled externally — the core's main.go
// calls plugin.Register(update.NewUpdatePlugin()).

// UpdatePlugin implements the plugin.Plugin interface for OS and package
// update management on Debian-based nodes.
type UpdatePlugin struct {
	svc            *Service
	schedule       string
	autoSecurity   bool
	securitySource string
}

// NewUpdatePlugin creates an UpdatePlugin with a shared Service instance.
func NewUpdatePlugin() *UpdatePlugin {
	svc := &Service{}
	svc.Init()
	return &UpdatePlugin{
		svc:            svc,
		schedule:       DefaultSchedule,
		autoSecurity:   DefaultAutoSecurity,
		securitySource: DefaultSecuritySource,
	}
}

func (p *UpdatePlugin) Name() string {
	return "update"
}

func (p *UpdatePlugin) Version() string {
	return "0.1.0"
}

func (p *UpdatePlugin) Description() string {
	return "OS and package update management"
}

func (p *UpdatePlugin) Routes() http.Handler {
	return newRouter(p.svc, p.CurrentConfig)
}

func (p *UpdatePlugin) ScheduledJobs() []plugin.JobDefinition {
	if !p.autoSecurity {
		return nil
	}
	if p.securitySource == "available" && !p.svc.SecurityAvailable() {
		return nil
	}
	return []plugin.JobDefinition{
		{
			ID:          "update.security",
			Description: "Run automatic security updates",
			Cron:        p.schedule,
			Func:        p.svc.RunSecurityUpdates,
		},
	}
}

func (p *UpdatePlugin) Endpoints() []plugin.Endpoint {
	return []plugin.Endpoint{
		{Method: http.MethodGet, Path: "/status", Description: "Pending updates and system info"},
		{Method: http.MethodGet, Path: "/logs", Description: "Last update run output"},
		{Method: http.MethodGet, Path: "/config", Description: "Update plugin configuration"},
		{Method: http.MethodPost, Path: "/run", Description: "Trigger update run"},
	}
}

// Configure applies persisted config from the YAML file at startup.
func (p *UpdatePlugin) Configure(cfg map[string]any) {
	if cfg == nil {
		return
	}
	if v, ok := cfg["schedule"].(string); ok && v != "" {
		p.schedule = v
	}
	if v, ok := cfg["auto_security"].(bool); ok {
		p.autoSecurity = v
	}
	if v, ok := cfg["security_source"].(string); ok && (v == "available" || v == "always") {
		p.securitySource = v
	}
}

// UpdateConfig validates and applies a single runtime config change.
func (p *UpdatePlugin) UpdateConfig(key string, value any) error {
	switch key {
	case "schedule":
		v, ok := value.(string)
		if !ok || v == "" {
			return fmt.Errorf("schedule must be a non-empty string")
		}
		p.schedule = v
	case "auto_security":
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("auto_security must be a boolean")
		}
		p.autoSecurity = v
	case "security_source":
		v, ok := value.(string)
		if !ok || v == "" {
			return fmt.Errorf("security_source must be a non-empty string")
		}
		if v != "available" && v != "always" {
			return fmt.Errorf("security_source must be 'available' or 'always'")
		}
		p.securitySource = v
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// CurrentConfig returns the plugin's current configuration.
func (p *UpdatePlugin) CurrentConfig() map[string]any {
	return map[string]any{
		"schedule":        p.schedule,
		"auto_security":   p.autoSecurity,
		"security_source": p.securitySource,
	}
}
