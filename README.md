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

### Windows Dependency Install Commands

Use one-time install commands before running `run.ps1`:

```powershell
winget install --id GoLang.Go -e
scoop install redis
# or:
choco install redis-64
```

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

Press `Ctrl+C` to stop all started server processes.

## Run (Windows PowerShell)

```powershell
.\run.ps1
```

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

- first 3 requests from the same client IP return `OK - served by :PORT`
- the 4th request within the same minute returns `429 Too Many Requests`

## Test

```bash
go test ./...
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```
