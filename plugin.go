package update

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

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
	DefaultSecuritySource = "detected"
)

// Note: Registration with the core is handled externally — the core's main.go
// calls plugin.Register(update.NewUpdatePlugin()).

// UpdatePlugin implements the plugin.Plugin interface for OS and package
// update management on Debian-based nodes.
type UpdatePlugin struct {
	svc *Service

	mu             sync.RWMutex
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
	p.mu.RLock()
	autoSec := p.autoSecurity
	secSrc := p.securitySource
	sched := p.schedule
	p.mu.RUnlock()

	// update.full is always available for manual triggering (no cron).
	jobs := []plugin.JobDefinition{
		{
			ID:          "update.full",
			Description: "Run full system upgrade",
			Func:        p.svc.RunFullUpgrade,
		},
	}

	// update.security is included when security_source="always" (regardless of
	// detection) or when security_source="detected" and the system actually has
	// a security apt source.  The cron schedule is attached only when
	// auto_security is enabled; otherwise the job is manual-trigger-only.
	secAvail := p.svc.SecurityAvailable()
	if secSrc == "detected" && !secAvail {
		return jobs
	}

	secJob := plugin.JobDefinition{
		ID:          "update.security",
		Description: "Run security updates",
		Func:        p.svc.RunSecurityUpdates,
	}
	if autoSec {
		secJob.Cron = sched
	}
	jobs = append(jobs, secJob)

	return jobs
}

func (p *UpdatePlugin) Endpoints() []plugin.Endpoint {
	return []plugin.Endpoint{
		{Method: http.MethodGet, Path: "/status", Description: "Pending updates and system info"},
		{Method: http.MethodGet, Path: "/logs", Description: "Last update run output"},
		{Method: http.MethodGet, Path: "/config", Description: "Update plugin configuration"},
		{Method: http.MethodPost, Path: "/run", Description: "Trigger update run"},
	}
}

// cronShortcuts are the standard @-shortcuts accepted by the core scheduler.
var cronShortcuts = map[string]bool{
	"@yearly":    true,
	"@annually":  true,
	"@monthly":   true,
	"@weekly":    true,
	"@daily":     true,
	"@midnight":  true,
	"@hourly":    true,
}

// validateCronExpr checks that expr is a valid cron expression structurally.
// It accepts the standard 5-field format and @-shortcuts (@daily, @weekly, etc.).
// Full semantic validation (field ranges, names) is performed by the core
// scheduler at job registration time.
func validateCronExpr(expr string) error {
	trimmed := strings.TrimSpace(expr)
	if cronShortcuts[strings.ToLower(trimmed)] {
		return nil
	}
	fields := strings.Fields(trimmed)
	if len(fields) != 5 {
		return fmt.Errorf(
			"expected 5 fields (minute hour dom month dow), got %d"+
				"; if your expression has a seconds field, remove it"+
				" (6/7-field cron is not supported)",
			len(fields))
	}
	return nil
}

// Configure applies persisted config from the YAML file at startup.
func (p *UpdatePlugin) Configure(cfg map[string]any) {
	if cfg == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if v, ok := cfg["schedule"].(string); ok && v != "" {
		if err := validateCronExpr(v); err != nil {
			slog.Warn("invalid schedule ignored",
				"plugin", "update", "schedule", v, "default", DefaultSchedule, "error", err)
		} else {
			p.schedule = strings.TrimSpace(v)
		}
	}
	if v, ok := cfg["auto_security"].(bool); ok {
		p.autoSecurity = v
	}
	if v, ok := cfg["security_source"].(string); ok && (v == "detected" || v == "always") {
		p.securitySource = v
	}
}

// UpdateConfig validates and applies a single runtime config change.
func (p *UpdatePlugin) UpdateConfig(key string, value any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch key {
	case "schedule":
		v, ok := value.(string)
		if !ok || v == "" {
			return fmt.Errorf("schedule must be a non-empty string")
		}
		if err := validateCronExpr(v); err != nil {
			return fmt.Errorf("invalid schedule: %w", err)
		}
		p.schedule = strings.TrimSpace(v)
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
		if v != "detected" && v != "always" {
			return fmt.Errorf("security_source must be 'detected' or 'always'")
		}
		p.securitySource = v
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// CurrentConfig returns the plugin's current configuration.
func (p *UpdatePlugin) CurrentConfig() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]any{
		"schedule":        p.schedule,
		"auto_security":   p.autoSecurity,
		"security_source": p.securitySource,
	}
}
