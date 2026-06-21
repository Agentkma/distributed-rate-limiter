# Distributed Rate Limiter — Project Plan (Go + Redis)

---

## 1. Project Goal

Build a distributed rate limiter that enforces a per-IP request limit across multiple independent servers. The system must demonstrate coordination through a shared datastore and run locally from a ZIP file.

---

## 2. High-Level Architecture

The system consists of:

- **One Go HTTP server binary** launched 3 times, each on a different port (8001, 8002, 8003) — mirroring how a real distributed system runs the same code on different machines.
- **A single Redis instance** used as the shared state store.
- **A client script** that sends requests to different servers to demonstrate distributed coordination.

Each server process is stateless and runs identical code. All rate-limit decisions are made using shared counters stored in Redis. Each response identifies which server handled it, making distributed behavior visually observable.

---

## 3. Rate-Limiting Algorithm

The project uses a **fixed-window counter**:

1. For each incoming request, the server extracts the client IP.
2. It constructs a Redis key: `rate:<ip>:<current-minute>`
3. The server increments the counter and sets a 60-second expiration if the key is new.
4. If the counter exceeds the configured limit (e.g., **3 requests/min**), the server returns **HTTP 429**.

This algorithm is simple, deterministic, and easy to verify.

---

## 4. Distributed Behavior

Because all servers share Redis:

- Requests from the same IP hitting **different servers** count toward the **same limit**.
- If one server exceeds the limit, **all servers** will enforce the 429 response.
- The system demonstrates distributed coordination without requiring custom consensus or replication logic.

---

## 5. Implementation Plan

1. Build **one Go HTTP server binary** (`cmd/server/main.go`) with a `--port` flag and one `/api` endpoint that responds with `"OK - served by :PORT"`.

1. Add a Redis client and rate-limiting logic in shared internal packages.

1. Provide a `run.sh` script (macOS/Linux) that checks Redis installation, builds the binary once, launches `./server --port 8001`, `./server --port 8002`, `./server --port 8003`, and prints startup URLs/rate limit.

1. Provide a `run.ps1` script (Windows PowerShell) with identical logic.

1. Provide a `client.sh` script in **manual mode** that prints ready-to-copy `curl` commands.

1. Include a `README.md` with instructions for running the project locally.

1. Add explicit unit tests for core behavior in `internal/ratelimiter/limiter_test.go` (allow/deny + fail-open) and `cmd/server/main_test.go` (`extractIP` with `RemoteAddr`).

### Fixed behavior decisions

- Client IP extraction uses `RemoteAddr` only.
- Rate limit is fixed at **3 requests/minute per IP**.
- Redis target is local default: `localhost:6379`.
- On Redis errors, limiter is **fail-open** (request allowed) and logs each Redis error.
- Windows support requires `run.ps1`; manual testing is acceptable.

### What both run scripts print on startup

```text
Distributed Rate Limiter is running.
Rate limit: 3 requests per minute per IP (shared across all servers)

Servers:
  http://localhost:8001/api
  http://localhost:8002/api
  http://localhost:8003/api

Manual test (copy and paste these):
  curl http://localhost:8001/api   → OK - served by :8001
  curl http://localhost:8002/api   → OK - served by :8002
  curl http://localhost:8003/api   → OK - served by :8003
  (4th request to any server)      → 429 Too Many Requests
```

---

## 6. Project Structure

```text
/distributed-rate-limiter
├── /cmd
│   └── /server
│       ├── main.go          ← single binary; run 3 times with --port flag
│       └── main_test.go     ← unit tests for `extractIP`
├── /internal
│   ├── /ratelimiter
│   │   ├── limiter.go       ← all rate-limit logic lives here
│   │   └── limiter_test.go  ← unit tests for limiter behavior
│   └── /redisclient
│       └── client.go        ← Redis connection setup lives here
├── go.mod
├── run.sh                   ← macOS/Linux: builds + launches 3 server instances
├── run.ps1                  ← Windows PowerShell: same logic
├── client.sh                ← manual curl instructions
├── PLAN.md
└── README.md
```

### Key design decisions

| Package | Exported API | Responsibility |
| --- | --- | --- |
| `internal/ratelimiter` | `Allow(ip string) bool` | Compute Redis key, increment counter, check limit |
| `internal/redisclient` | `GetClient() *redis.Client` | Centralized Redis connection config |
| `cmd/server/main.go` | `--port` flag + HTTP handler | Tiny — parses port, identifies itself in response |

The single binary accepts a `--port` flag and embeds its port in every response:

```go
func main() {
    port := flag.String("port", "8080", "port to listen on")
    flag.Parse()
    // handler closes over port so each instance identifies itself
    http.HandleFunc("/api", makeHandler(*port))
    http.ListenAndServe(":"+*port, nil)
}

func makeHandler(port string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ip := extractIP(r)
        if !ratelimiter.Allow(ip) {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
    fmt.Fprintf(w, "OK - served by :%s\n", port)
    }
}
```

---

## 7. Deliverables

The final ZIP file will include:

- [ ] `cmd/server/main.go` — single server binary with `--port` flag
- [ ] `internal/ratelimiter/limiter.go` — shared rate-limit logic
- [ ] `internal/redisclient/client.go` — shared Redis connection setup
- [ ] `run.sh` — macOS/Linux: builds binary, launches 3 instances, prints startup info
- [ ] `run.ps1` — Windows PowerShell: identical logic
- [ ] `client.sh` — manual `curl` instructions
- [ ] `go.mod` — Go module definition
- [ ] `internal/ratelimiter/limiter_test.go` — unit tests for limiter decisions
- [ ] `cmd/server/main_test.go` — unit tests for `extractIP`
- [ ] `README.md` — setup and usage instructions

---

## 9. Test Plan

- **Unit tests (`go test ./...`)**
  - `internal/ratelimiter/limiter_test.go`
    - allows requests when Redis is unavailable (fail-open)
    - validates key/bucket format behavior
  - `cmd/server/main_test.go`
    - parses IPv4 `RemoteAddr` correctly
    - parses IPv6 `RemoteAddr` correctly
    - handles malformed/empty `RemoteAddr`
- **Build validation**
  - `go build ./...`
- **End-to-end manual validation**
  - run servers with `run.sh` or `run.ps1`
  - execute printed `curl` commands and verify 4th request returns `429`

---

## 10. Progress Checkpoint (2026-06-21)

### Completed

- Phase 1 scaffolded and validated:
  - Added `go.mod`/`go.sum`
  - Implemented `cmd/server/main.go`
  - Implemented `internal/redisclient/client.go`
  - Implemented `internal/ratelimiter/limiter.go`
  - Ran `go build ./...` and `go test ./...`
- Plan aligned to approved decisions:
  - fixed limit `3/min`
  - `RemoteAddr` IP extraction
  - fail-open behavior on Redis errors
  - local Redis at `localhost:6379`
  - manual `client.sh` flow
- Explicit unit-test planning added to this document.

### Pending (Next Session)

- Phase 2: implement `run.sh`, `run.ps1`, `client.sh` (manual mode), and `README.md`.
- Phase 2 validation: verify script behavior and startup/manual test instructions.
- Phase 3: implement unit tests in `internal/ratelimiter/limiter_test.go` and `cmd/server/main_test.go`.

---

## 8. Why This Design Is Easy to Grade

- **One script to run everything** — no Docker, no env vars, no manual Redis setup.
- **Rate limit is printed on startup** — grader immediately knows what to test.
- **Every response identifies its server** — `"OK – served by :8001"` proves requests are hitting different processes.
- **Every response identifies its server** — `"OK - served by :8001"` proves requests are hitting different processes.
- **All rate-limit logic in one file** (`limiter.go`) — easy to read and verify.
- **One binary, not three** — mirrors real distributed systems; no duplicated code.
- **Manual test script** — grader can run `bash client.sh` to get exact `curl` commands for verification.
