package update

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// defaultConfigFn returns a config function with default values for tests.
func defaultConfigFn() func() map[string]any {
	return func() map[string]any {
		return map[string]any{
			"schedule":        DefaultSchedule,
			"auto_security":   DefaultAutoSecurity,
			"security_source": DefaultSecuritySource,
		}
	}
}

func TestHandleStatus(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/status", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json", ct)
	}
}

func TestHandleRun_MissingBody(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/run", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_EmptyType(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	body := `{"type": ""}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString(body))
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_UnknownType(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	body := `{"type": "partial"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString(body))
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_InvalidJSON(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString("not json"))
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_BodyTooLarge(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	// Create a body larger than maxRequestBody (1 MB)
	large := `{"type": "` + strings.Repeat("x", maxRequestBody+1) + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString(large))
	router.ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandleLogs(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/logs", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var status RunStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if status.Status != "none" {
		t.Fatalf("got status %q, want %q", status.Status, "none")
	}
}

func TestHandleConfig(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/config", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var cfg map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := cfg["schedule"]; !ok {
		t.Fatal("missing schedule in config response")
	}
	if _, ok := cfg["auto_security"]; !ok {
		t.Fatal("missing auto_security in config response")
	}
	if _, ok := cfg["security_source"]; !ok {
		t.Fatal("missing security_source in config response")
	}
	if _, ok := cfg["security_available"]; !ok {
		t.Fatal("missing security_available in config response")
	}
}

func TestHandleConfig_SecurityAvailableReflectsService(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc, defaultConfigFn())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/config", nil)
	router.ServeHTTP(w, r)

	var cfg map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if cfg["security_available"] != false {
		t.Errorf("expected security_available=false, got %v", cfg["security_available"])
	}

	svc2 := &Service{securityAvailable: true}
	router2 := newRouter(svc2, defaultConfigFn())
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/config", nil)
	router2.ServeHTTP(w2, r2)

	var cfg2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &cfg2); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if cfg2["security_available"] != true {
		t.Errorf("expected security_available=true, got %v", cfg2["security_available"])
	}
}
