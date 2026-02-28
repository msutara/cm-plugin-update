package update

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msutara/config-manager-core/plugin"
)

// Compile-time interface compliance (redundant with plugin.go but tests the
// assertion independently).
var _ plugin.Plugin = (*UpdatePlugin)(nil)

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

func TestUpdatePlugin_Endpoints(t *testing.T) {
	p := NewUpdatePlugin()
	eps := p.Endpoints()

	if len(eps) != 4 {
		t.Fatalf("Endpoints: got %d, want 4", len(eps))
	}

	want := []struct{ method, path string }{
		{http.MethodGet, "/status"},
		{http.MethodGet, "/logs"},
		{http.MethodGet, "/config"},
		{http.MethodPost, "/run"},
	}
	for i, w := range want {
		if eps[i].Method != w.method || eps[i].Path != w.path {
			t.Errorf("endpoint[%d] = %s %s, want %s %s", i, eps[i].Method, eps[i].Path, w.method, w.path)
		}
		if eps[i].Description == "" {
			t.Errorf("endpoint[%d] has empty description", i)
		}
	}
}

func TestUpdatePlugin_ScheduledJobs(t *testing.T) {
	p := NewUpdatePlugin()
	jobs := p.ScheduledJobs()

	if !p.svc.SecurityAvailable() {
		// On non-Linux or systems without a security apt source, the
		// security cron job is omitted.
		if len(jobs) != 0 {
			t.Fatalf("ScheduledJobs: expected empty when security unavailable, got %d", len(jobs))
		}
		return
	}

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
