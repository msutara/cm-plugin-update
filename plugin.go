package update

import (
	"net/http"

	"github.com/msutara/cm-plugin-update/pluginiface"
)

// Compile-time check: UpdatePlugin implements pluginiface.Plugin.
var _ pluginiface.Plugin = (*UpdatePlugin)(nil)

// Note: Registration with the core is handled externally — the core's main.go
// calls plugin.Register(update.NewUpdatePlugin()). This plugin uses a local
// pluginiface mirror of the core interface for independent development.

// UpdatePlugin implements the pluginiface.Plugin interface for OS and package
// update management on Debian-based nodes.
type UpdatePlugin struct {
	svc *Service
}

// NewUpdatePlugin creates an UpdatePlugin with a shared Service instance.
func NewUpdatePlugin() *UpdatePlugin {
	return &UpdatePlugin{svc: &Service{}}
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

func (p *UpdatePlugin) ScheduledJobs() []pluginiface.JobDefinition {
	return []pluginiface.JobDefinition{
		{
			ID:          "update.security",
			Description: "Run automatic security updates",
			Cron:        "0 3 * * *",
			Func:        p.svc.RunSecurityUpdates,
		},
	}
}
