package update

import (
	"net/http"

	"github.com/msutara/config-manager-core/plugin"
)

// Compile-time check: UpdatePlugin implements plugin.Plugin.
var _ plugin.Plugin = (*UpdatePlugin)(nil)

// Note: Registration with the core is handled externally — the core's main.go
// calls plugin.Register(update.NewUpdatePlugin()).

// UpdatePlugin implements the plugin.Plugin interface for OS and package
// update management on Debian-based nodes.
type UpdatePlugin struct {
	svc *Service
}

// NewUpdatePlugin creates an UpdatePlugin with a shared Service instance.
func NewUpdatePlugin() *UpdatePlugin {
	svc := &Service{}
	svc.Init()
	return &UpdatePlugin{svc: svc}
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
	return newRouter(p.svc)
}

func (p *UpdatePlugin) ScheduledJobs() []plugin.JobDefinition {
	if !p.svc.SecurityAvailable() {
		return nil
	}
	return []plugin.JobDefinition{
		{
			ID:          "update.security",
			Description: "Run automatic security updates",
			Cron:        "0 3 * * *",
			Func:        p.svc.RunSecurityUpdates,
		},
	}
}
