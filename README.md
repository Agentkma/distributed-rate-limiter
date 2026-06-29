# Distributed Rate Limiter (Go + Redis)

A local demo of a distributed, per-IP fixed-window rate limiter.

Three identical server processes share Redis, so requests counted on one server affect limits enforced by the others.

## Supported Systems

- macOS (using `run.sh`)
- Windows PowerShell (using `run.ps1`)

Linux is not currently a supported run-script target for this project.

## Prerequisites

- Go 1.22+
- Redis running locally on `localhost:6379`
- `redis-cli` available in your PATH

### macOS Dependency Install Commands

Use one-time install/start commands before running `run.sh`:

```bash
brew install go
brew install redis
brew services start redis
```

If Redis was just started, the first connectivity check can fail briefly while it finishes booting. If that happens, wait a few seconds and run `bash run.sh` again.

If ports `8001`, `8002`, or `8003` are already in use, `run.sh` will stop before startup and tell you which port is occupied.

### Windows Dependency Install Commands

Use one-time install commands before running `run.ps1`:

```powershell
winget install --id GoLang.Go -e
scoop install redis
# or:
choco install redis-64
```

If Redis was just started, the first connectivity check can fail briefly while it finishes booting. If that happens, wait a few seconds and run `./run.ps1` again.

If ports `8001`, `8002`, or `8003` are already in use, `run.ps1` will stop before startup and tell you which port is occupied.

## Run (macOS)

This path is supported for macOS.

```bash
bash run.sh
```

This script:

- verifies Redis is installed and reachable
- builds the server binary once
- starts server instances on ports 8001, 8002, and 8003
- prints manual curl commands
- streams server logs directly in the same terminal without creating log files in the project folder

Important: keep that terminal running while servers are up. Run `curl` tests from a second terminal window/tab.

Press `Ctrl+C` to stop all started server processes.

## Shutdown / Cleanup

When you are done testing:

- Stop server processes started by `run.sh` or `run.ps1` with `Ctrl+C` in the same terminal.
- If Redis was started with Homebrew on macOS, stop it with:

```bash
brew services stop redis
```

- If Redis was started manually, stop it with:

```bash
redis-cli -h localhost -p 6379 shutdown
```

## Run (Windows PowerShell)

```powershell
.\run.ps1
```

Keep that terminal running while servers are up. Run `curl` tests from a second terminal window/tab.

Press `Ctrl+C` to stop all started server processes.

## Manual Request Test

```bash
bash client.sh
```

Or run directly:

```bash
curl http://localhost:8001/api
curl http://localhost:8002/api
curl http://localhost:8003/api
curl http://localhost:8001/api
```

Expected behavior:

- the limit is 3 requests per minute per client IP total across all servers (not 3 per server)
- first 3 requests from the same client IP return `OK - served by :PORT`
- the 4th request within the same minute returns `429 Too Many Requests`

## Test

```bash
go test ./...
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```
