$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptRoot

$redisAddr = "localhost:6379"
$redisHost, $redisPort = $redisAddr.Split(':')

$serverPath = Join-Path $scriptRoot "server.exe"
$ports = @(8001, 8002, 8003)
$processes = @()

function Try-Command {
    param(
        [scriptblock]$Command
    )

    try {
        & $Command *> $null
        return ($LASTEXITCODE -eq 0 -or $null -eq $LASTEXITCODE)
    }
    catch {
        return $false
    }
}

function Command-Exists {
    param(
        [string]$Name
    )

    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Wait-For-Redis {
    param(
        [int]$Retries,
        [int]$DelaySeconds
    )

    for ($attempt = 1; $attempt -le $Retries; $attempt++) {
        if (Try-Command { redis-cli -h $redisHost -p $redisPort ping }) {
            return $true
        }

        Start-Sleep -Seconds $DelaySeconds
    }

    return $false
}

function Port-Is-In-Use {
    param(
        [int]$Port
    )

    return Try-Command { lsof -nP -iTCP:$Port -sTCP:LISTEN }
}

function Write-Lines {
    param(
        [string[]]$Lines
    )

    foreach ($line in $Lines) {
        Write-Host $line
    }
}

function Fail-With-Help {
    param(
        [string[]]$Lines,
        [string]$ErrorMessage
    )

    Write-Lines -Lines $Lines
    Write-Error $ErrorMessage
}

if (-not (Command-Exists "go")) {
    Fail-With-Help -Lines @(
        "go is not installed or not in PATH."
        "This script supports Windows. Install Go with:"
        "  winget install --id GoLang.Go -e"
    ) -ErrorMessage "Missing dependency: go"
}

if (-not (Command-Exists "redis-cli")) {
    Fail-With-Help -Lines @(
        "redis-cli is not installed or not in PATH."
        "This script supports Windows. Install Redis with one of these options:"
        "  scoop install redis"
        "  choco install redis-64"
    ) -ErrorMessage "Missing dependency: redis-cli"
}

if (-not (Wait-For-Redis -Retries 5 -DelaySeconds 1)) {
    Fail-With-Help -Lines @(
        "Redis is not reachable at $redisAddr."
        "If Redis was just started, wait a few seconds and run this script again."
        "Try starting Redis with one of these:"
        "  redis-server"
        "  Start-Service Redis"
        "Then run this script again (expected address: $redisAddr)."
    ) -ErrorMessage "Redis startup check failed"
}

foreach ($port in $ports) {
    if (Port-Is-In-Use -Port $port) {
        Fail-With-Help -Lines @(
            "Port $port is already in use."
            "Stop the existing process using that port and run this script again."
            "To inspect it, run:"
            "  lsof -nP -iTCP:$port -sTCP:LISTEN"
        ) -ErrorMessage "Port preflight check failed"
    }
}

# Build the server binary once; it will be launched on multiple ports below.
go build -o $serverPath ./cmd/server

# Start one server per port in the background and track processes for cleanup.
foreach ($port in $ports) {
    $proc = Start-Process -FilePath $serverPath -ArgumentList "--port", "$port" -PassThru
    $processes += $proc
}

Write-Lines -Lines @(
    "Distributed Rate Limiter is running."
    "Rate limit: 3 requests per minute per IP (shared across all servers)"
    ""
    "Servers:"
    "  http://localhost:8001/api"
    "  http://localhost:8002/api"
    "  http://localhost:8003/api"
    ""
    "Manual test (copy and paste these):"
    "  Keep this terminal running. Open a second terminal for curl tests."
    "  curl http://localhost:8001/api   -> OK - served by :8001"
    "  curl http://localhost:8002/api   -> OK - served by :8002"
    "  curl http://localhost:8003/api   -> OK - served by :8003"
    "  (4th request to any server)      -> 429 Too Many Requests"
    ""
    "Press Ctrl+C to stop all servers."
)

try {
    while ($true) {
        Start-Sleep -Seconds 1
    }
}
finally {
    foreach ($proc in $processes) {
        if (-not $proc.HasExited) {
            Stop-Process -Id $proc.Id -Force
        }
    }
}