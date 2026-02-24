package update

import (
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestParsePendingUpdates_Typical(t *testing.T) {
	output := `Listing... Done
curl/bookworm-security 8.4.0-2+b1 amd64 [upgradable from: 8.4.0-2]
vim/bookworm 9.0.1378-2 amd64 [upgradable from: 9.0.1378-1]
libssl3/bookworm-security 3.0.11-1~deb12u1 amd64 [upgradable from: 3.0.9-1]
`
	updates := parsePendingUpdates(output)
	if len(updates) != 3 {
		t.Fatalf("expected 3 updates, got %d", len(updates))
	}

	cases := []struct {
		pkg, oldVer, newVer string
		security            bool
	}{
		{"curl", "8.4.0-2", "8.4.0-2+b1", true},
		{"vim", "9.0.1378-1", "9.0.1378-2", false},
		{"libssl3", "3.0.9-1", "3.0.11-1~deb12u1", true},
	}
	for i, tc := range cases {
		u := updates[i]
		if u.Package != tc.pkg {
			t.Errorf("[%d] package: got %q, want %q", i, u.Package, tc.pkg)
		}
		if u.CurrentVersion != tc.oldVer {
			t.Errorf("[%d] current_version: got %q, want %q", i, u.CurrentVersion, tc.oldVer)
		}
		if u.NewVersion != tc.newVer {
			t.Errorf("[%d] new_version: got %q, want %q", i, u.NewVersion, tc.newVer)
		}
		if u.Security != tc.security {
			t.Errorf("[%d] security: got %v, want %v", i, u.Security, tc.security)
		}
	}
}

func TestParsePendingUpdates_EmptyOutput(t *testing.T) {
	updates := parsePendingUpdates("")
	if len(updates) != 0 {
		t.Fatalf("expected 0 updates, got %d", len(updates))
	}
}

func TestParsePendingUpdates_ListingOnly(t *testing.T) {
	updates := parsePendingUpdates("Listing... Done\n")
	if len(updates) != 0 {
		t.Fatalf("expected 0 updates, got %d", len(updates))
	}
}

func TestParsePendingUpdates_MalformedLines(t *testing.T) {
	output := `Listing... Done
this line has no slash
also/bad
curl/bookworm-security 8.4.0-2+b1 amd64 [upgradable from: 8.4.0-2]
`
	updates := parsePendingUpdates(output)
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	if updates[0].Package != "curl" {
		t.Errorf("package: got %q, want %q", updates[0].Package, "curl")
	}
}

func TestParsePendingUpdates_NoUpgradableFromMarker(t *testing.T) {
	output := "nginx/bookworm 1.22.1-9 amd64\n"
	updates := parsePendingUpdates(output)
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	if updates[0].CurrentVersion != "" {
		t.Errorf("expected empty current_version, got %q", updates[0].CurrentVersion)
	}
}

func TestGetLastRunStatus_Default(t *testing.T) {
	svc := &Service{}
	status, err := svc.GetLastRunStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "none" {
		t.Errorf("status: got %q, want %q", status.Status, "none")
	}
	if status.Type != "" {
		t.Errorf("type: got %q, want empty", status.Type)
	}
	if status.StartedAt != nil {
		t.Errorf("started_at: expected nil, got %v", status.StartedAt)
	}
}

func TestServiceInit(t *testing.T) {
	svc := &Service{}
	if svc.lastRun != nil {
		t.Error("expected lastRun to be nil on new Service")
	}

	// Verify ListPendingUpdates works (returns empty on non-Linux).
	updates, err := svc.ListPendingUpdates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updates == nil {
		t.Error("expected non-nil slice from ListPendingUpdates")
	}
}

func TestRunSecurityUpdates_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping non-Linux test on Linux")
	}
	svc := &Service{}
	err := svc.RunSecurityUpdates()
	if !errors.Is(err, errNotLinux) {
		t.Fatalf("got %v, want errNotLinux", err)
	}
}

func TestRunFullUpgrade_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping non-Linux test on Linux")
	}
	svc := &Service{}
	err := svc.RunFullUpgrade()
	if !errors.Is(err, errNotLinux) {
		t.Fatalf("got %v, want errNotLinux", err)
	}
}

func TestParseUpgradedCount(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   int
	}{
		{
			"typical",
			"5 upgraded, 2 newly installed, 0 to remove and 3 not upgraded.",
			7,
		},
		{
			"zero",
			"0 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.",
			0,
		},
		{
			"no match",
			"Reading package lists... Done",
			0,
		},
		{
			"empty",
			"",
			0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseUpgradedCount(tc.output)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestParsePendingUpdates_ReturnsEmptySliceNotNil(t *testing.T) {
	updates := parsePendingUpdates("")
	if updates == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
}

func TestGetLastRunStatus_DefensiveCopy(t *testing.T) {
	svc := &Service{}
	now := time.Now()
	svc.lastRun = &RunStatus{
		Type:      "security",
		Status:    "success",
		StartedAt: &now,
		Duration:  "1.5s",
		Packages:  3,
		Log:       "test log",
	}

	s1, err := svc.GetLastRunStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the returned copy
	s1.Status = "mutated"
	s1.Packages = 999
	*s1.StartedAt = time.Time{}

	// Original must be unchanged
	s2, err := svc.GetLastRunStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s2.Status != "success" {
		t.Errorf("status mutated: got %q, want %q", s2.Status, "success")
	}
	if s2.Packages != 3 {
		t.Errorf("packages mutated: got %d, want %d", s2.Packages, 3)
	}
	if s2.StartedAt.IsZero() {
		t.Error("startedAt mutated to zero")
	}
}

func TestListPendingUpdates_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping non-Linux test on Linux")
	}
	svc := &Service{}
	updates, err := svc.ListPendingUpdates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updates == nil {
		t.Fatal("expected non-nil empty slice on non-Linux")
	}
	if len(updates) != 0 {
		t.Fatalf("expected 0 updates on non-Linux, got %d", len(updates))
	}
}

func TestRunAptCommand_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping non-Linux test on Linux")
	}
	svc := &Service{}
	err := svc.runAptCommand("test", "version")
	if !errors.Is(err, errNotLinux) {
		t.Fatalf("got %v, want errNotLinux", err)
	}
}

func TestParsePendingUpdates_MultipleSecuritySources(t *testing.T) {
	output := `Listing... Done
openssl/bookworm-security 3.0.11-1 amd64 [upgradable from: 3.0.9-1]
git/bookworm 1:2.39.5-0+deb12u1 amd64 [upgradable from: 1:2.39.2-1.1]
zlib1g/bookworm-security 1:1.2.13.dfsg-1 amd64 [upgradable from: 1:1.2.13-1]
`
	updates := parsePendingUpdates(output)
	if len(updates) != 3 {
		t.Fatalf("expected 3 updates, got %d", len(updates))
	}

	securityCount := 0
	for _, u := range updates {
		if u.Security {
			securityCount++
		}
	}
	if securityCount != 2 {
		t.Errorf("expected 2 security updates, got %d", securityCount)
	}
}

func TestParseUpgradedCount_LargeNumbers(t *testing.T) {
	got := parseUpgradedCount("125 upgraded, 13 newly installed, 2 to remove and 0 not upgraded.")
	if got != 138 {
		t.Errorf("got %d, want 138", got)
	}
}
