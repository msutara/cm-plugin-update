package update

import (
	"net/http"

	"github.com/msutara/cm-plugin-update/pluginiface"
)

// Compile-time check: UpdatePlugin implements pluginiface.Plugin.
var _ pluginiface.Plugin = (*UpdatePlugin)(nil)

// Note: The specification (specs/SPEC.md) describes self-registration via
// plugin.Register() in an init() function. That call is intentionally omitted
// here because pluginiface is a local mirror of the core interface for
// independent development. Registration will be wired when integrating with
// config-manager-core.

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
