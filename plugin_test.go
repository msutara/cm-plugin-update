package update

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msutara/config-manager-core/plugin"
)

// Compile-time interface compliance (redundant with plugin.go but tests the
// assertion independently).
var (
	_ plugin.Plugin       = (*UpdatePlugin)(nil)
	_ plugin.Configurable = (*UpdatePlugin)(nil)
)

// findJob returns the job with the given ID, or nil if not found.
func findJob(jobs []plugin.JobDefinition, id string) *plugin.JobDefinition {
	for i := range jobs {
		if jobs[i].ID == id {
			return &jobs[i]
		}
	}
	return nil
}

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

	// update.full is always present.
	full := findJob(jobs, "update.full")
	if full == nil {
		t.Fatal("ScheduledJobs should always include update.full")
	}
	if full.Cron != "" {
		t.Errorf("update.full should have empty cron (manual only), got %q", full.Cron)
	}
	if full.Func == nil {
		t.Error("update.full Func is nil")
	}

	if p.svc.SecurityAvailable() {
		// On Linux with security source, update.security should also be present.
		sec := findJob(jobs, "update.security")
		if sec == nil {
			t.Fatal("expected update.security when security available")
		}
		if sec.Cron == "" {
			t.Error("update.security should have cron when auto_security=true (default)")
		}
		if sec.Func == nil {
			t.Error("update.security Func is nil")
		}
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
	sec := findJob(jobs, "update.security")
	if sec == nil {
		t.Fatal("update.security job not found in scheduled jobs")
	}
	if sec.Cron != "0 22 * * 5" {
		t.Errorf("security job cron: got %q, want %q", sec.Cron, "0 22 * * 5")
	}
}

func TestUpdatePlugin_ScheduledJobsDisabledWhenAutoSecurityFalse(t *testing.T) {
	p := NewUpdatePlugin()
	p.autoSecurity = false
	p.svc = newTestService(true)

	jobs := p.ScheduledJobs()
	// update.full always present; update.security present but without cron.
	full := findJob(jobs, "update.full")
	if full == nil {
		t.Error("expected update.full job to be present")
	}
	sec := findJob(jobs, "update.security")
	if sec == nil {
		t.Error("expected update.security job to be present (unscheduled with empty cron)")
	} else if sec.Cron != "" {
		t.Error("update.security should have no cron when auto_security=false")
	}
}

func TestUpdatePlugin_ScheduledJobsAlwaysMode(t *testing.T) {
	p := NewUpdatePlugin()
	p.autoSecurity = true
	p.securitySource = "always"
	p.svc = newTestService(false)

	jobs := p.ScheduledJobs()
	if findJob(jobs, "update.full") == nil {
		t.Error("expected update.full job to be present")
	}
	if findJob(jobs, "update.security") == nil {
		t.Fatal("securitySource=always should include update.security even when SecurityAvailable()=false")
	}
}

func TestValidateCronExpr_Valid(t *testing.T) {
	for _, expr := range []string{"0 3 * * *", "*/5 * * * *", "0 2 * * MON"} {
		if err := validateCronExpr(expr); err != nil {
			t.Errorf("validateCronExpr(%q) = %v, want nil", expr, err)
		}
	}
}

func TestValidateCronExpr_Shortcuts(t *testing.T) {
	for _, expr := range []string{"@daily", "@weekly", "@monthly", "@yearly", "@annually", "@midnight", "@hourly"} {
		if err := validateCronExpr(expr); err != nil {
			t.Errorf("validateCronExpr(%q) = %v, want nil", expr, err)
		}
	}
}

func TestValidateCronExpr_ShortcutsCaseInsensitive(t *testing.T) {
	for _, expr := range []string{"@Daily", "@WEEKLY", "@Monthly"} {
		if err := validateCronExpr(expr); err != nil {
			t.Errorf("validateCronExpr(%q) = %v, want nil", expr, err)
		}
	}
}

func TestValidateCronExpr_TooManyFields(t *testing.T) {
	err := validateCronExpr("0 2 * * * MON")
	if err == nil {
		t.Fatal("expected error for 6-field cron")
	}
	if !strings.Contains(err.Error(), "expected 5 fields") {
		t.Errorf("error should mention 5 fields: %v", err)
	}
	if !strings.Contains(err.Error(), "seconds field") {
		t.Errorf("error should mention seconds field: %v", err)
	}
}

func TestValidateCronExpr_TooFewFields(t *testing.T) {
	err := validateCronExpr("0 3 *")
	if err == nil {
		t.Fatal("expected error for 3-field cron")
	}
	if !strings.Contains(err.Error(), "got 3") {
		t.Errorf("error should say got 3: %v", err)
	}
}

func TestUpdatePlugin_ConfigureInvalidCronKeepsDefault(t *testing.T) {
	p := NewUpdatePlugin()
	original := p.schedule
	p.Configure(map[string]any{
		"schedule": "0 2 * * * MON",
	})
	if p.schedule != original {
		t.Errorf("invalid cron should not change schedule: got %q, want %q", p.schedule, original)
	}
}

func TestUpdatePlugin_UpdateConfigInvalidCronReturnsError(t *testing.T) {
	p := NewUpdatePlugin()
	err := p.UpdateConfig("schedule", "0 2 * * * MON")
	if err == nil {
		t.Fatal("expected error for 6-field cron in UpdateConfig")
	}
	if !strings.Contains(err.Error(), "invalid schedule") {
		t.Errorf("error should say 'invalid schedule': %v", err)
	}
}

func TestUpdatePlugin_UpdateConfigValidCronAccepted(t *testing.T) {
	p := NewUpdatePlugin()
	err := p.UpdateConfig("schedule", "0 2 * * 1")
	if err != nil {
		t.Fatalf("valid 5-field cron should succeed: %v", err)
	}
	if p.schedule != "0 2 * * 1" {
		t.Errorf("schedule = %q, want %q", p.schedule, "0 2 * * 1")
	}
}

func TestUpdatePlugin_UpdateConfigShortcutAccepted(t *testing.T) {
	p := NewUpdatePlugin()
	err := p.UpdateConfig("schedule", "@daily")
	if err != nil {
		t.Fatalf("@daily shortcut should succeed: %v", err)
	}
	if p.schedule != "@daily" {
		t.Errorf("schedule = %q, want %q", p.schedule, "@daily")
	}
}

func TestUpdatePlugin_Configure6FieldKeepsDefaultSchedule(t *testing.T) {
	// Regression test: 6-field Quartz cron should not persist, and
	// ScheduledJobs should use the default (not the invalid expression).
	p := NewUpdatePlugin()
	p.autoSecurity = true
	p.securitySource = "always"
	p.svc = newTestService(true)

	p.Configure(map[string]any{
		"schedule": "0 2 * * * MON",
	})
	if p.schedule != DefaultSchedule {
		t.Errorf("schedule = %q after invalid cron, want default %q", p.schedule, DefaultSchedule)
	}

	jobs := p.ScheduledJobs()
	secJob := findJob(jobs, "update.security")
	if secJob == nil {
		t.Fatal("update.security job should be present")
	}
	if secJob.Cron != DefaultSchedule {
		t.Errorf("security job cron = %q, want default %q", secJob.Cron, DefaultSchedule)
	}
}

func TestUpdatePlugin_ScheduledJobsDetectedMode(t *testing.T) {
	p := NewUpdatePlugin()
	p.autoSecurity = true
	p.securitySource = "detected"
	p.svc = newTestService(false)

	jobs := p.ScheduledJobs()
	// update.full always present; update.security omitted when detected + unavailable.
	if findJob(jobs, "update.full") == nil {
		t.Error("expected update.full job to be present")
	}
	if findJob(jobs, "update.security") != nil {
		t.Error("securitySource=detected should omit update.security when SecurityAvailable()=false")
	}
	if jobs[0].ID != "update.full" {
		t.Errorf("expected update.full, got %q", jobs[0].ID)
	}
}

func TestUpdatePlugin_ConfigureTrimsWhitespace(t *testing.T) {
	p := NewUpdatePlugin()
	p.Configure(map[string]any{
		"schedule": "  @daily  ",
	})
	cfg := p.CurrentConfig()
	if cfg["schedule"] != "@daily" {
		t.Errorf("Configure should trim whitespace: got %q", cfg["schedule"])
	}
}

func TestUpdatePlugin_UpdateConfigTrimsWhitespace(t *testing.T) {
	p := NewUpdatePlugin()
	if err := p.UpdateConfig("schedule", " 0 5 * * * "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := p.CurrentConfig()
	if cfg["schedule"] != "0 5 * * *" {
		t.Errorf("UpdateConfig should trim whitespace: got %q", cfg["schedule"])
	}
}
