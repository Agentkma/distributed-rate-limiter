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

1. Build **one Go HTTP server binary** (`cmd/server/main.go`) with:
   - A `--port` flag so the same binary runs on any port.
   - One endpoint (`/api`) that responds with `"OK – served by :PORT"`, identifying itself in every response.
2. Add a Redis client and rate-limiting logic in shared internal packages.
3. Provide a `run.sh` script (macOS/Linux) that:
   - Checks if Redis is installed and exits with a clear message if not.
   - Builds the binary once.
   - Launches 3 instances: `./server --port 8001`, `./server --port 8002`, `./server --port 8003`.
   - Prints the configured rate limit and all server URLs on startup.
4. Provide a `run.ps1` script (Windows PowerShell) with identical logic.
5. Provide a `client.sh` script with two modes:
   - **Manual mode:** prints ready-to-copy `curl` commands the grader can run one at a time.
   - **Auto mode:** fires requests automatically across all 3 servers, printing a timestamped log of each request, which server responded, and the HTTP status — making the 429 moment unmistakable.
6. Include a `README.md` with instructions for running the project locally.

### What both run scripts print on startup

```text
Distributed Rate Limiter is running.
Rate limit: 3 requests per minute per IP (shared across all servers)

Servers:
  http://localhost:8001/api
  http://localhost:8002/api
  http://localhost:8003/api

Manual test (copy and paste these):
  curl http://localhost:8001/api   → OK – served by :8001
  curl http://localhost:8002/api   → OK – served by :8002
  curl http://localhost:8003/api   → OK – served by :8003
  (4th request to any server)      → 429 Too Many Requests

Or run the automated client:
  bash client.sh
```

### What `client.sh` auto mode prints

```text
[1] GET :8001  →  200  "OK – served by :8001"
[2] GET :8002  →  200  "OK – served by :8002"
[3] GET :8003  →  200  "OK – served by :8003"
[4] GET :8001  →  429  "Too Many Requests"
[5] GET :8002  →  429  "Too Many Requests"

Rate limit hit after 3 requests. All servers are enforcing the shared limit.
```

---

## 6. Project Structure

```text
/distributed-rate-limiter
├── /cmd
│   └── /server
│       └── main.go          ← single binary; run 3 times with --port flag
├── /internal
│   ├── /ratelimiter
│   │   └── limiter.go       ← all rate-limit logic lives here
│   └── /redisclient
│       └── client.go        ← Redis connection setup lives here
├── go.mod
├── run.sh                   ← macOS/Linux: builds + launches 3 server instances
├── run.ps1                  ← Windows PowerShell: same logic
├── client.sh                ← manual curl instructions + automated test mode
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
        fmt.Fprintf(w, "OK – served by :%s\n", port)
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
- [ ] `client.sh` — manual `curl` instructions + automated timestamped test log
- [ ] `go.mod` — Go module definition
- [ ] `README.md` — setup and usage instructions

---

## 8. Why This Design Is Easy to Grade

- **One script to run everything** — no Docker, no env vars, no manual Redis setup.
- **Rate limit is printed on startup** — grader immediately knows what to test.
- **Every response identifies its server** — `"OK – served by :8001"` proves requests are hitting different processes.
- **All rate-limit logic in one file** (`limiter.go`) — easy to read and verify.
- **One binary, not three** — mirrors real distributed systems; no duplicated code.
- **Automated client script** — grader can run `bash client.sh` and watch the 429 appear with a clear log, without typing a single `curl` command.
