package update

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// maxRequestBody is the maximum allowed size for incoming request bodies (1 MB).
const maxRequestBody = 1 << 20

func newRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	h := &handler{svc: svc}
	r.Get("/status", h.handleStatus)
	r.Post("/run", h.handleRun)
	r.Get("/logs", h.handleLogs)
	r.Get("/config", h.handleConfig)
	return r
}

// handler groups HTTP handlers with a shared Service instance.
type handler struct {
	svc *Service
}

func (h *handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	updates, err := h.svc.ListPendingUpdates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list updates", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updates)
}

func (h *handler) handleRun(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	var req struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "missing update type", "type must be 'security' or 'full'")
			return
		}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "missing update type", "type must be 'security' or 'full'")
		return
	}

	var err error
	switch req.Type {
	case "security":
		err = h.svc.RunSecurityUpdates()
	case "full":
		err = h.svc.RunFullUpgrade()
	default:
		writeError(w, http.StatusBadRequest, "invalid update type", "type must be 'security' or 'full'")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started", "type": req.Type})
}

func (h *handler) handleLogs(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetLastRunStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get logs", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *handler) handleConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := map[string]any{
		"auto_security_updates": true,
		"schedule":              "0 3 * * *",
	}
	writeJSON(w, http.StatusOK, cfg)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, code int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": details,
		},
	}); err != nil {
		slog.Error("failed to write error response", "error", err)
	}
}
