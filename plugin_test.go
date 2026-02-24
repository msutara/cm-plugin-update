package update

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msutara/cm-plugin-update/pluginiface"
)

// Compile-time interface compliance (redundant with plugin.go but tests the
// assertion independently).
var _ pluginiface.Plugin = (*UpdatePlugin)(nil)

func TestNewUpdatePlugin(t *testing.T) {
	p := NewUpdatePlugin()
	if p == nil {
		t.Fatal("NewUpdatePlugin returned nil")
	}
	if p.svc == nil {
		t.Fatal("NewUpdatePlugin().svc is nil")
	}
}

func TestUpdatePlugin_Metadata(t *testing.T) {
	p := NewUpdatePlugin()

	if got := p.Name(); got != "update" {
		t.Errorf("Name: got %q, want %q", got, "update")
	}
	if got := p.Version(); got == "" {
		t.Error("Version: got empty string")
	}
	if got := p.Description(); got == "" {
		t.Error("Description: got empty string")
	}
}

func TestUpdatePlugin_Routes(t *testing.T) {
	p := NewUpdatePlugin()
	h := p.Routes()
	if h == nil {
		t.Fatal("Routes returned nil handler")
	}

	// Smoke test: the router should respond to known routes.
	// Use /config (no subprocess side effects) rather than /status
	// which calls apt on Linux and may fail in constrained environments.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/config", nil)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("/config: got %d, want %d", w.Code, http.StatusOK)
	}

	// Unknown routes should 405 or 404
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	h.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Error("/nonexistent: expected non-200 for unknown route")
	}
}

func TestUpdatePlugin_ScheduledJobs(t *testing.T) {
	p := NewUpdatePlugin()
	jobs := p.ScheduledJobs()
	if len(jobs) == 0 {
		t.Fatal("ScheduledJobs returned empty slice")
	}

	job := jobs[0]
	if job.ID == "" {
		t.Error("job ID is empty")
	}
	if job.Description == "" {
		t.Error("job Description is empty")
	}
	if job.Cron == "" {
		t.Error("job Cron is empty")
	}
	if job.Func == nil {
		t.Error("job Func is nil")
	}
}
