package update

import (
	"net/http"

	"github.com/msutara/cm-plugin-update/pluginiface"
)

// Compile-time check: UpdatePlugin implements pluginiface.Plugin.
var _ pluginiface.Plugin = (*UpdatePlugin)(nil)

// UpdatePlugin implements the pluginiface.Plugin interface for OS and package
// update management on Debian-based nodes.
type UpdatePlugin struct{}

func (p *UpdatePlugin) Name() string        { return "update" }
func (p *UpdatePlugin) Version() string     { return "0.1.0" }
func (p *UpdatePlugin) Description() string { return "OS and package update management" }

func (p *UpdatePlugin) Routes() http.Handler {
	return newRouter()
}

func (p *UpdatePlugin) ScheduledJobs() []pluginiface.JobDefinition {
	svc := &Service{}
	return []pluginiface.JobDefinition{
		{
			ID:          "update.security",
			Description: "Run automatic security updates",
			Cron:        "0 3 * * *",
			Func:        svc.RunSecurityUpdates,
		},
	}
}
