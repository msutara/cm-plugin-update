package update

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msutara/config-manager-core/plugin"
)

// Compile-time interface compliance (redundant with plugin.go but tests the
// assertion independently).
var (
	_ plugin.Plugin       = (*UpdatePlugin)(nil)
	_ plugin.Configurable = (*UpdatePlugin)(nil)
)

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

// --- Configurable interface tests ---

func TestUpdatePlugin_ConfigureNil(t *testing.T) {
	p := NewUpdatePlugin()
	p.Configure(nil)
	// Should retain defaults.
	cfg := p.CurrentConfig()
	if cfg["schedule"] != DefaultSchedule {
		t.Errorf("schedule: got %v, want %v", cfg["schedule"], DefaultSchedule)
	}
	if cfg["auto_security"] != DefaultAutoSecurity {
		t.Errorf("auto_security: got %v, want %v", cfg["auto_security"], DefaultAutoSecurity)
	}
	if cfg["security_source"] != DefaultSecuritySource {
		t.Errorf("security_source: got %v, want %v", cfg["security_source"], DefaultSecuritySource)
	}
}

func TestUpdatePlugin_ConfigureAppliesValues(t *testing.T) {
	p := NewUpdatePlugin()
	p.Configure(map[string]any{
		"schedule":        "0 5 * * 1",
		"auto_security":   false,
		"security_source": "always",
	})
	cfg := p.CurrentConfig()
	if cfg["schedule"] != "0 5 * * 1" {
		t.Errorf("schedule: got %v, want '0 5 * * 1'", cfg["schedule"])
	}
	if cfg["auto_security"] != false {
		t.Errorf("auto_security: got %v, want false", cfg["auto_security"])
	}
	if cfg["security_source"] != "always" {
		t.Errorf("security_source: got %v, want 'always'", cfg["security_source"])
	}
}

func TestUpdatePlugin_ConfigureIgnoresUnknownKeys(t *testing.T) {
	p := NewUpdatePlugin()
	p.Configure(map[string]any{
		"unknown_key": "whatever",
		"schedule":    "0 6 * * *",
	})
	cfg := p.CurrentConfig()
	if cfg["schedule"] != "0 6 * * *" {
		t.Errorf("schedule: got %v, want '0 6 * * *'", cfg["schedule"])
	}
	if _, exists := cfg["unknown_key"]; exists {
		t.Error("unknown_key should not appear in CurrentConfig")
	}
}

func TestUpdatePlugin_ConfigureIgnoresWrongTypes(t *testing.T) {
	p := NewUpdatePlugin()
	p.Configure(map[string]any{
		"schedule":        123,   // wrong type — should be ignored
		"auto_security":   "yes", // wrong type — should be ignored
		"security_source": 123,   // wrong type — should be ignored
	})
	cfg := p.CurrentConfig()
	if cfg["schedule"] != DefaultSchedule {
		t.Errorf("schedule: got %v, want default %v", cfg["schedule"], DefaultSchedule)
	}
	if cfg["auto_security"] != DefaultAutoSecurity {
		t.Errorf("auto_security: got %v, want default %v", cfg["auto_security"], DefaultAutoSecurity)
	}
	if cfg["security_source"] != DefaultSecuritySource {
		t.Errorf("security_source: got %v, want default %v", cfg["security_source"], DefaultSecuritySource)
	}
}

func TestUpdatePlugin_UpdateConfigSchedule(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("schedule", "0 4 * * *"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.schedule != "0 4 * * *" {
		t.Errorf("schedule: got %q, want %q", p.schedule, "0 4 * * *")
	}
}

func TestUpdatePlugin_UpdateConfigAutoSecurity(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("auto_security", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.autoSecurity != false {
		t.Error("auto_security should be false")
	}
}

func TestUpdatePlugin_UpdateConfigSecuritySource(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("security_source", "always"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.securitySource != "always" {
		t.Errorf("security_source: got %q, want %q", p.securitySource, "always")
	}
}

func TestUpdatePlugin_UpdateConfigSecuritySourceInvalid(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("security_source", "never"); err == nil {
		t.Error("expected error for invalid security_source")
	}
}

func TestUpdatePlugin_UpdateConfigUnknownKey(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("nonexistent", "value"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestUpdatePlugin_UpdateConfigWrongType(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("schedule", 123); err == nil {
		t.Error("expected error for non-string schedule")
	}
	if err := p.UpdateConfig("auto_security", "yes"); err == nil {
		t.Error("expected error for non-bool auto_security")
	}
	if err := p.UpdateConfig("security_source", 123); err == nil {
		t.Error("expected error for non-string security_source")
	}
}

func TestUpdatePlugin_UpdateConfigEmptySchedule(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("schedule", ""); err == nil {
		t.Error("expected error for empty schedule")
	}
}

func TestUpdatePlugin_UpdateConfigEmptySecuritySource(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("security_source", ""); err == nil {
		t.Error("expected error for empty security_source")
	}
}

func TestUpdatePlugin_ConfigureIgnoresEmptyStrings(t *testing.T) {
	p := NewUpdatePlugin()
	p.Configure(map[string]any{
		"schedule":        "",
		"security_source": "",
	})
	cfg := p.CurrentConfig()
	if cfg["schedule"] != DefaultSchedule {
		t.Errorf("empty schedule should be ignored, got %v", cfg["schedule"])
	}
	if cfg["security_source"] != DefaultSecuritySource {
		t.Errorf("empty security_source should be ignored, got %v", cfg["security_source"])
	}
}

func TestUpdatePlugin_ConfigureIgnoresInvalidSecuritySource(t *testing.T) {
	p := NewUpdatePlugin()
	p.Configure(map[string]any{
		"security_source": "never",
	})
	if p.securitySource != DefaultSecuritySource {
		t.Errorf("invalid security_source should be ignored, got %q", p.securitySource)
	}
}

func TestUpdatePlugin_CurrentConfigReturnsNewMap(t *testing.T) {
	p := NewUpdatePlugin()
	cfg1 := p.CurrentConfig()
	cfg2 := p.CurrentConfig()
	cfg1["schedule"] = "mutated"
	if cfg2["schedule"] == "mutated" {
		t.Error("CurrentConfig should return independent maps")
	}
}

func TestUpdatePlugin_ScheduledJobsUsesConfigSchedule(t *testing.T) {
	p := NewUpdatePlugin()
	p.schedule = "0 22 * * 5"
	p.autoSecurity = true
	p.securitySource = "always"

	jobs := p.ScheduledJobs()
	if len(jobs) == 0 {
		t.Fatal("expected at least one job")
	}
	if jobs[0].Cron != "0 22 * * 5" {
		t.Errorf("job cron: got %q, want %q", jobs[0].Cron, "0 22 * * 5")
	}
}

func TestUpdatePlugin_ScheduledJobsDisabledWhenAutoSecurityFalse(t *testing.T) {
	p := NewUpdatePlugin()
	p.autoSecurity = false
	p.svc.securityAvailable = true

	jobs := p.ScheduledJobs()
	if len(jobs) != 0 {
		t.Errorf("expected no jobs when auto_security=false, got %d", len(jobs))
	}
}

func TestUpdatePlugin_ScheduledJobsAlwaysMode(t *testing.T) {
	p := NewUpdatePlugin()
	p.autoSecurity = true
	p.securitySource = "always"
	p.svc.securityAvailable = false

	jobs := p.ScheduledJobs()
	if len(jobs) == 0 {
		t.Fatal("securitySource=always should schedule even when SecurityAvailable()=false")
	}
}

func TestUpdatePlugin_ScheduledJobsAvailableMode(t *testing.T) {
	p := NewUpdatePlugin()
	p.autoSecurity = true
	p.securitySource = "available"
	p.svc.securityAvailable = false

	jobs := p.ScheduledJobs()
	if len(jobs) != 0 {
		t.Errorf("securitySource=available should skip when SecurityAvailable()=false, got %d", len(jobs))
	}
}
