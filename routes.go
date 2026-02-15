package update

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func newRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/status", handleStatus)
	r.Post("/run", handleRun)
	r.Get("/logs", handleLogs)
	r.Get("/config", handleConfig)
	return r
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	svc := &Service{}
	updates, err := svc.ListPendingUpdates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list updates", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updates)
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	svc := &Service{}
	var err error
	switch req.Type {
	case "security":
		err = svc.RunSecurityUpdates()
	case "full":
		err = svc.RunFullUpgrade()
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

func handleLogs(w http.ResponseWriter, r *http.Request) {
	svc := &Service{}
	status, err := svc.GetLastRunStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get logs", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func handleConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := map[string]any{
		"auto_security_updates": true,
		"schedule":              "0 3 * * *",
	}
	writeJSON(w, http.StatusOK, cfg)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, code int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}
