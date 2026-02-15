package update

import (
	"net/http"

	"github.com/msutara/config-manager-core/internal/plugin"
)

// UpdatePlugin implements the plugin.Plugin interface for OS and package
// update management on Debian-based nodes.
type UpdatePlugin struct{}

func init() {
	plugin.Register(&UpdatePlugin{})
}

func (p *UpdatePlugin) Name() string        { return "update" }
func (p *UpdatePlugin) Version() string     { return "0.1.0" }
func (p *UpdatePlugin) Description() string { return "OS and package update management" }

func (p *UpdatePlugin) Routes() http.Handler {
	return newRouter()
}

func (p *UpdatePlugin) ScheduledJobs() []plugin.JobDefinition {
	svc := &Service{}
	return []plugin.JobDefinition{
		{
			ID:          "update.security",
			Description: "Run automatic security updates",
			Cron:        "0 3 * * *",
			Func:        svc.RunSecurityUpdates,
		},
	}
}
