package update

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PendingUpdate represents a single package that has an update available.
type PendingUpdate struct {
	Package        string `json:"package"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	Security       bool   `json:"security"`
}

// RunStatus represents the outcome of the last update run.
type RunStatus struct {
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	Duration  string     `json:"duration"`
	Packages  int        `json:"packages"`
	Log       string     `json:"log"`
}

// Service contains the domain logic for update management.
type Service struct {
	mu      sync.Mutex
	lastRun *RunStatus
}

var (
	errNotLinux    = errors.New("update plugin requires Linux")
	errAptNotFound = errors.New("apt-get not found in PATH")
)

// parsePendingUpdates parses the output of `apt list --upgradable` into
// PendingUpdate structs. Each output line has the form:
//
//	package/source version_new arch [upgradable from: version_old]
func parsePendingUpdates(output string) []PendingUpdate {
	updates := make([]PendingUpdate, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Listing") {
			continue
		}

		slashIdx := strings.Index(line, "/")
		if slashIdx < 0 {
			continue
		}
		pkg := line[:slashIdx]
		rest := line[slashIdx+1:]

		fields := strings.Fields(rest)
		if len(fields) < 2 {
			continue
		}
		source := fields[0]
		newVersion := fields[1]
		security := strings.Contains(source, "-security")

		var oldVersion string
		const marker = "[upgradable from: "
		if idx := strings.Index(line, marker); idx >= 0 {
			start := idx + len(marker)
			if end := strings.Index(line[start:], "]"); end >= 0 {
				oldVersion = line[start : start+end]
			}
		}

		updates = append(updates, PendingUpdate{
			Package:        pkg,
			CurrentVersion: oldVersion,
			NewVersion:     newVersion,
			Security:       security,
		})
	}
	return updates
}

// ListPendingUpdates queries the system for available package upgrades.
func (s *Service) ListPendingUpdates() ([]PendingUpdate, error) {
	if runtime.GOOS != "linux" {
		slog.Info("apt not available, skipping update check", "plugin", "update", "os", runtime.GOOS)
		return []PendingUpdate{}, nil
	}

	aptPath, err := exec.LookPath("apt")
	if err != nil {
		slog.Info("apt not found in PATH, skipping update check", "plugin", "update")
		return []PendingUpdate{}, nil
	}

	cmd := exec.Command(aptPath, "list", "--upgradable")
	cmd.Env = append(cmd.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("apt list --upgradable failed: %w: %s", err, string(out))
	}

	updates := parsePendingUpdates(string(out))
	slog.Info("listed pending updates", "plugin", "update", "count", len(updates))
	return updates, nil
}

// upgradeCountRe matches the apt-get summary line:
//
//	"N upgraded, M newly installed, P to remove and Q not upgraded."
var upgradeCountRe = regexp.MustCompile(`(\d+)\s+upgraded,\s*(\d+)\s+newly installed`)

// parseUpgradedCount extracts the total number of upgraded + newly installed
// packages from apt-get combined output.
func parseUpgradedCount(output string) int {
	m := upgradeCountRe.FindStringSubmatch(output)
	if m == nil {
		return 0
	}
	upgraded, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	installed, err := strconv.Atoi(m[2])
	if err != nil {
		return upgraded
	}
	return upgraded + installed
}

// runAptCommand executes an apt-get command and records the result in lastRun.
func (s *Service) runAptCommand(runType string, args ...string) error {
	if runtime.GOOS != "linux" {
		return errNotLinux
	}

	aptGetPath, err := exec.LookPath("apt-get")
	if err != nil {
		return errAptNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()
	slog.Info("starting update run", "plugin", "update", "type", runType)

	cmd := exec.Command(aptGetPath, args...)
	cmd.Env = append(cmd.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.CombinedOutput()

	duration := time.Since(start)
	status := &RunStatus{
		Type:      runType,
		Status:    "success",
		StartedAt: &start,
		Duration:  duration.Round(time.Millisecond).String(),
		Packages:  parseUpgradedCount(string(out)),
		Log:       string(out),
	}

	if err != nil {
		status.Status = "failed"
		slog.Error("update run failed", "plugin", "update", "type", runType, "error", err)
	} else {
		slog.Info("update run completed", "plugin", "update", "type", runType, "duration", duration)
	}

	s.lastRun = status

	if err != nil {
		// Include truncated output in error for better diagnostics.
		detail := string(out)
		if len(detail) > 500 {
			detail = detail[len(detail)-500:]
		}
		return fmt.Errorf("%s failed: %w: %s", runType, err, detail)
	}
	return nil
}

// RunSecurityUpdates applies only security pocket updates by restricting
// the apt target release to the distribution's security pocket.
func (s *Service) RunSecurityUpdates() error {
	if runtime.GOOS != "linux" {
		return errNotLinux
	}

	codename, err := distroCodename()
	if err != nil {
		return fmt.Errorf("cannot determine distribution codename: %w", err)
	}

	return s.runAptCommand("security",
		"-y", "-o", "Dpkg::Options::=--force-confold",
		"-t", codename+"-security",
		"upgrade",
	)
}

// distroCodename reads the VERSION_CODENAME from /etc/os-release.
func distroCodename() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "VERSION_CODENAME="))
			val = strings.Trim(val, `"'`)
			if val != "" {
				return val, nil
			}
		}
	}
	return "", errors.New("VERSION_CODENAME not found in /etc/os-release")
}

// RunFullUpgrade applies all pending package upgrades.
func (s *Service) RunFullUpgrade() error {
	if runtime.GOOS != "linux" {
		return errNotLinux
	}
	return s.runAptCommand("full",
		"-y", "-o", "Dpkg::Options::=--force-confold", "dist-upgrade",
	)
}

// GetLastRunStatus returns the outcome of the most recent update run.
// Returns a defensive copy so callers cannot mutate internal state.
func (s *Service) GetLastRunStatus() (*RunStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastRun == nil {
		return &RunStatus{
			Status: "none",
		}, nil
	}

	cp := *s.lastRun
	if s.lastRun.StartedAt != nil {
		t := *s.lastRun.StartedAt
		cp.StartedAt = &t
	}
	return &cp, nil
}
