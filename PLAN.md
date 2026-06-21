# Distributed Rate Limiter — Project Plan (Go + Redis)

---

## 1. Project Goal

Build a distributed rate limiter that enforces a per-IP request limit across multiple independent servers. The system must demonstrate coordination through a shared datastore and run locally from a ZIP file.

---

## 2. High-Level Architecture

The system consists of:

- **Multiple Go HTTP servers** (3–5 instances), each running on a different port.
- **A single Redis instance** used as the shared state store.
- **A client script** that sends requests to random servers to demonstrate distributed behavior.

Each server is stateless and identical. All rate-limit decisions are made using shared counters stored in Redis.

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

1. Implement a small Go HTTP server with one endpoint (`/api`).
2. Add a Redis client and rate-limiting logic.
3. Add command-line flags so each server can run on a different port.
4. Provide a `run.sh` script (macOS/Linux) that:
   - Checks if Redis is installed and exits with a clear message if not.
   - Starts Redis in the background.
   - Starts all Go servers (ports 8001, 8002, 8003).
   - Prints the configured rate limit and server URLs so the grader knows exactly how to test.
5. Provide a `run.ps1` script (Windows PowerShell) with identical logic.
6. Provide a client script to generate test traffic.
7. Include a `README.md` with instructions for running the project locally.

### What both run scripts print on startup

```text
Distributed Rate Limiter is running.
Rate limit: 3 requests per minute per IP (shared across all servers)

Servers:
  http://localhost:8001/api
  http://localhost:8002/api
  http://localhost:8003/api

Try sending requests:
  curl http://localhost:8001/api
  curl http://localhost:8002/api   ← same IP limit applies here too
```

---

## 6. Project Structure

```text
/distributed-rate-limiter
├── /cmd
│   ├── /server1
│   │   └── main.go
│   ├── /server2
│   │   └── main.go
│   └── /server3
│       └── main.go
├── /internal
│   ├── /ratelimiter
│   │   └── limiter.go       ← all rate-limit logic lives here
│   └── /redisclient
│       └── client.go        ← Redis connection setup lives here
├── go.mod
├── run.sh                   ← macOS/Linux startup script
├── run.ps1                  ← Windows PowerShell startup script
├── client.sh                ← test traffic generator
├── PLAN.md
└── README.md
```

### Key design decisions

| Package | Exported API | Responsibility |
| --- | --- | --- |
| `internal/ratelimiter` | `Allow(ip string) bool` | Compute Redis key, increment counter, check limit |
| `internal/redisclient` | `GetClient() *redis.Client` | Centralized Redis connection config |
| `cmd/server*/main.go` | HTTP handler only | Tiny — just calls `ratelimiter.Allow()` |

Each server's handler becomes:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ip := extractIP(r)
    if !ratelimiter.Allow(ip) {
        http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
        return
    }
    fmt.Fprintln(w, "OK")
}
```

---

## 7. Deliverables

The final ZIP file will include:

- [ ] Go source code for all servers (`/cmd/`)
- [ ] Shared rate-limiter package (`/internal/ratelimiter/`)
- [ ] Shared Redis client package (`/internal/redisclient/`)
- [ ] `run.sh` — macOS/Linux startup script
- [ ] `run.ps1` — Windows PowerShell startup script
- [ ] `client.sh` — test traffic generator script
- [ ] `go.mod` — Go module definition
- [ ] `README.md` — setup and usage instructions

---

## 8. Why This Design Is Easy to Grade

- **One script to run everything** — no Docker, no env vars, no manual Redis setup.
- **Rate limit is printed on startup** — grader immediately knows what to test.
- **All rate-limit logic in one file** (`limiter.go`) — easy to read and verify.
- **Servers are tiny** — grader can confirm each server shares the same logic at a glance.
- **Distributed behavior is observable** — hitting different ports with the same IP triggers 429 on all of them.
