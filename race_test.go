package update

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestConcurrentConfigAccess verifies that concurrent reads and writes
// to plugin configuration do not race.  Run with -race to detect issues.
func TestConcurrentConfigAccess(t *testing.T) {
	p := NewUpdatePlugin()
	var wg sync.WaitGroup

	// Writer: UpdateConfig
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.UpdateConfig("schedule", "0 4 * * *")
			_ = p.UpdateConfig("auto_security", false)
			_ = p.UpdateConfig("security_source", "always")
		}()
	}

	// Reader: CurrentConfig
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := p.CurrentConfig()
			if cfg == nil {
				t.Error("CurrentConfig returned nil")
			}
		}()
	}

	// Reader: ScheduledJobs
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.ScheduledJobs()
		}()
	}

	wg.Wait()
}

// TestConcurrentConfigureAndRead verifies that Configure (bulk write)
// does not race with CurrentConfig (read).
func TestConcurrentConfigureAndRead(t *testing.T) {
	p := NewUpdatePlugin()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.Configure(map[string]any{
				"schedule":        "0 5 * * 1",
				"auto_security":   false,
				"security_source": "always",
			})
		}()
		go func() {
			defer wg.Done()
			_ = p.CurrentConfig()
		}()
	}

	wg.Wait()
}

// TestConcurrentRunRejection verifies that the running guard rejects
// a second call. The running guard fires before LookPath, so we can
// exercise it on any platform by manipulating the unexported running flag.
func TestConcurrentRunRejection(t *testing.T) {
	svc := &Service{}

	// Directly verify the guard logic: set running, attempt runAptCommand.
	svc.mu.Lock()
	svc.running = true
	svc.mu.Unlock()

	err := svc.runAptCommand("test", "version")
	if runtime.GOOS != "linux" {
		// Non-Linux: OS check fires before running guard.
		if !errors.Is(err, errNotLinux) {
			t.Fatalf("got %v, want errNotLinux", err)
		}
	} else {
		// Linux: running guard fires before LookPath.
		if !errors.Is(err, errAlreadyRunning) {
			t.Fatalf("got %v, want errAlreadyRunning", err)
		}
	}

	// Verify the flag is still set (guard did not clear it).
	svc.mu.Lock()
	stillRunning := svc.running
	svc.mu.Unlock()
	if !stillRunning {
		t.Error("running flag was incorrectly cleared")
	}

	// Clean up.
	svc.mu.Lock()
	svc.running = false
	svc.mu.Unlock()
}

// TestInitIdempotent verifies Init can be called multiple times safely.
func TestInitIdempotent(t *testing.T) {
	svc := &Service{}
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.Init()
		}()
	}

	wg.Wait()

	// After all inits, SecurityAvailable should be deterministic.
	val := svc.SecurityAvailable()
	for i := 0; i < 10; i++ {
		if svc.SecurityAvailable() != val {
			t.Fatal("SecurityAvailable returned inconsistent value after Init")
		}
	}
}

// TestConcurrentGetLastRunStatus verifies that reading last run status
// concurrently with writes does not race.
func TestConcurrentGetLastRunStatus(t *testing.T) {
	svc := &Service{}
	var wg sync.WaitGroup

	// Writer: simulate runAptCommand updating lastRun.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			now := time.Now()
			status := &RunStatus{
				Type:      "test",
				Status:    "success",
				StartedAt: &now,
				Duration:  "1s",
				Packages:  n,
				Log:       "log",
			}
			svc.mu.Lock()
			svc.lastRun = status
			svc.mu.Unlock()
		}(i)
	}

	// Concurrent readers.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := svc.GetLastRunStatus()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if status == nil {
				t.Error("GetLastRunStatus returned nil")
			}
		}()
	}

	wg.Wait()
}
