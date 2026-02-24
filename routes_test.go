package update

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleStatus(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc)
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
	router := newRouter(svc)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/run", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_EmptyType(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc)
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
	router := newRouter(svc)
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
	router := newRouter(svc)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString("not json"))
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_BodyTooLarge(t *testing.T) {
	svc := &Service{}
	router := newRouter(svc)
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
	router := newRouter(svc)
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
	router := newRouter(svc)
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
	if _, ok := cfg["auto_security_updates"]; !ok {
		t.Fatal("missing auto_security_updates in config response")
	}
	if _, ok := cfg["schedule"]; !ok {
		t.Fatal("missing schedule in config response")
	}
}
