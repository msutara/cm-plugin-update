package update

import "time"

// PendingUpdate represents a single package that has an update available.
type PendingUpdate struct {
	Package        string `json:"package"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	Security       bool   `json:"security"`
}

// RunStatus represents the outcome of the last update run.
type RunStatus struct {
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	Duration  string    `json:"duration"`
	Packages  int       `json:"packages"`
	Log       string    `json:"log"`
}

// Service contains the domain logic for update management.
type Service struct{}

// ListPendingUpdates queries the system for available package upgrades.
func (s *Service) ListPendingUpdates() ([]PendingUpdate, error) {
	// TODO: shell out to apt to list upgradable packages
	return []PendingUpdate{}, nil
}

// RunSecurityUpdates applies only security pocket updates.
func (s *Service) RunSecurityUpdates() error {
	// TODO: run apt-get upgrade with security-only filter
	return nil
}

// RunFullUpgrade applies all pending package upgrades.
func (s *Service) RunFullUpgrade() error {
	// TODO: run apt-get dist-upgrade
	return nil
}

// GetLastRunStatus returns the outcome of the most recent update run.
func (s *Service) GetLastRunStatus() (*RunStatus, error) {
	// TODO: read persisted run status from disk
	return &RunStatus{
		Status: "none",
	}, nil
}
