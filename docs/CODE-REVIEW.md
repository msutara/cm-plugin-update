# Code Review — cm-plugin-update

Reviewed files: `plugin.go`, `routes.go`, `service.go`, `service_test.go`,
`pluginiface/pluginiface.go`, `specs/SPEC.md`, `docs/USAGE.md`.

Severity scale: **Critical** → **High** → **Medium** → **Low** → **Informational**

---

## Findings

### 1. `parsePendingUpdates` returns `nil` slice — JSON `null` instead of `[]` (High) ✅ Fixed

**File:** `service.go` — `parsePendingUpdates`

**Problem:** The function declared `var updates []PendingUpdate`, which is a
`nil` slice. When no updates are found the nil slice is passed to
`json.NewEncoder(w).Encode`, which serializes it as `null` instead of `[]`.
API consumers that iterate the array would need to guard against `null`.

**Fix applied:** Changed the declaration to `updates := make([]PendingUpdate, 0)`
so an empty slice is always returned and encodes as `[]`.
---

### 2. `RunFullUpgrade` missing early non-Linux guard (Medium) ✅ Fixed

**File:** `service.go` — `RunFullUpgrade`

**Problem:** `RunSecurityUpdates` guards against non-Linux platforms at the top
of the function and returns `errAptNotAvailable` immediately. `RunFullUpgrade`
relied on the inner `runAptCommand` to perform the same check. This was not
wrong at runtime, but the inconsistency made the two exported functions behave
differently from the caller's perspective (one returned the error directly; the
other went through an extra stack frame) and obscured intent.

**Fix applied:** Added the same `runtime.GOOS != "linux"` early-return guard to
`RunFullUpgrade`, matching the pattern used in `RunSecurityUpdates`.

---

### 3. `RunStatus.Packages` is always 0 (Medium)

**File:** `service.go` — `runAptCommand`, `RunStatus`

**Problem:** The `Packages` field of `RunStatus` is declared and serialized as
part of the `/logs` response, but it is never populated. Callers always receive
`"packages": 0`, regardless of how many packages were actually updated.

**Correction:** Parse the `apt-get` combined output for the summary line and
extract the count. For `dist-upgrade` and `upgrade` the output contains a line
of the form:
```
N upgraded, M newly installed, P to remove and Q not upgraded.
```

Sum `N` (upgraded) and `M` (newly installed) and assign the result to
`status.Packages` before storing `s.lastRun`.

---

### 4. `handleRun` blocks indefinitely during `apt-get` (Medium)

**File:** `routes.go` — `handleRun`

**Problem:** `RunSecurityUpdates` and `RunFullUpgrade` each shell out to
`apt-get`, which can take several minutes. The HTTP handler waits synchronously
for the command to finish, tying up the goroutine and holding the connection
open until the update completes (or the client disconnects). On the Raspberry Pi
target hardware, a full `dist-upgrade` over a slow connection can take 10–30
minutes.

**Correction (two options):**

*Option A — fire-and-forget job:* Return `202 Accepted` immediately, run the
apt command in a goroutine, and let the caller poll `/logs` for completion. Add
a `"running"` status to `RunStatus` set at job start and cleared on finish.

*Option B — streaming response:* Stream the `apt-get` stdout/stderr back to the
HTTP client in real time using chunked transfer encoding, so the caller can
observe progress without a hard timeout.

Option A is simpler and recommended; it also aligns naturally with the existing
`GetLastRunStatus` endpoint.

---

### 5. `handleConfig` returns hardcoded values (Low)

**File:** `routes.go` — `handleConfig`

**Problem:** The configuration map is hardcoded in the handler:

```go
cfg := map[string]any{
    "auto_security_updates": true,
    "schedule":              "0 3 * * *",
}
```

Any change to the schedule requires a code change and redeployment. There is no
persistence layer and no way for a caller to update the configuration.

**Correction:** Introduce a `Config` struct held by `Service` (with mutex
protection), expose a `SetConfig` / `GetConfig` method, and add a `PATCH
/config` route. Defaults can remain the same. Persistence can be deferred to a
future phase.

---

### 6. `errAptNotAvailable` conflates two distinct conditions (Low)

**File:** `service.go`

**Problem:** The sentinel error `errAptNotAvailable` is returned both when the
runtime OS is not Linux and when `apt-get` is not found in `$PATH`. These are
meaningfully different: the first means the plugin will never work; the second
means the binary is simply missing and might be installed later.

**Correction:** Define two separate sentinel errors:

```go
var (
    errNotLinux      = errors.New("update plugin requires Linux")
    errAptNotFound   = errors.New("apt-get not found in PATH")
)
```

Update callers and tests accordingly.

---

### 7. No HTTP handler tests (Low)

**File:** `service_test.go` (missing `routes_test.go`)

**Problem:** All existing tests cover only service-layer logic. The HTTP
handlers in `routes.go` (request parsing, error responses, JSON serialization)
have no test coverage.

**Correction:** Add `routes_test.go` using `net/http/httptest`. At minimum:

- `POST /run` with missing body → 400 with correct error JSON
- `POST /run` with `type: "security"` / `"full"` on a stub service → 200
- `POST /run` with unknown type → 400
- `GET /status` happy path and service error path
- `GET /logs` happy path
- `GET /config` happy path
- Body exceeding 1 MB → 413

---

### 8. Concurrent update runs are silently serialized (Informational)

**File:** `service.go` — `runAptCommand`

**Problem:** `s.mu.Lock()` is acquired inside `runAptCommand`, so a second
`/run` request will block until the first apt-get call finishes. The caller
receives no indication that an update is already in progress; they simply wait.

**Correction:** Check for an in-progress run at the start of `runAptCommand`
and return an explicit error (e.g., `errUpdateInProgress`) so the HTTP handler
can respond with `409 Conflict`.

---

### 9. `apt list --upgradable` result not refreshed before run (Informational)

**File:** `service.go` — `RunSecurityUpdates`, `RunFullUpgrade`

**Problem:** Neither run function calls `apt-get update` before applying
upgrades, so the local package cache may be stale. A freshly imaged system will
apply no updates if the cache has never been populated.

**Correction:** Prepend an `apt-get update -qq` call (or use
`-o Acquire::ForceIPv4=true` for reliability) before the upgrade command, or
document this as a prerequisite.

---

## Summary

| # | Severity      | File        | Status      |
|---|---------------|-------------|-------------|
| 1 | High          | service.go  | ✅ Fixed    |
| 2 | Medium        | service.go  | ✅ Fixed    |
| 3 | Medium        | service.go  | Pending     |
| 4 | Medium        | routes.go   | Pending     |
| 5 | Low           | routes.go   | Pending     |
| 6 | Low           | service.go  | Pending     |
| 7 | Low           | (missing)   | Pending     |
| 8 | Informational | service.go  | Pending     |
| 9 | Informational | service.go  | Pending     |
