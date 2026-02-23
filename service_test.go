package update

import (
	"errors"
	"runtime"
	"testing"
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
	if !errors.Is(err, errAptNotAvailable) {
		t.Fatalf("got %v, want errAptNotAvailable", err)
	}
}

func TestRunFullUpgrade_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping non-Linux test on Linux")
	}
	svc := &Service{}
	err := svc.RunFullUpgrade()
	if !errors.Is(err, errAptNotAvailable) {
		t.Fatalf("got %v, want errAptNotAvailable", err)
	}
}
