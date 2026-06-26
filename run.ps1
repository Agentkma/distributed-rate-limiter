$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptRoot

$redisAddr = "localhost:6379"
$redisHost, $redisPort = $redisAddr.Split(':')

$serverPath = Join-Path $scriptRoot "server.exe"
$ports = @(8001, 8002, 8003)
$processes = @()

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "go is not installed or not in PATH."
    Write-Host "Install Go with:"
    Write-Host "  winget install --id GoLang.Go -e"
    Write-Error "Missing dependency: go"
}

if (-not (Get-Command redis-cli -ErrorAction SilentlyContinue)) {
    Write-Host "redis-cli is not installed or not in PATH."
    Write-Host "Install Redis CLI with one of these options:"
    Write-Host "  scoop install redis"
    Write-Host "  choco install redis-64"
    Write-Error "Missing dependency: redis-cli"
}

$pingResult = & redis-cli -h $redisHost -p $redisPort ping 2>$null
if ($LASTEXITCODE -ne 0 -or $pingResult -notmatch "PONG") {
    Write-Error "Redis is not reachable at $redisAddr. Start Redis locally and run again."
}

go build -o $serverPath ./cmd/server

foreach ($port in $ports) {
    $proc = Start-Process -FilePath $serverPath -ArgumentList "--port", "$port" -PassThru
    $processes += $proc
}

Write-Host "Distributed Rate Limiter is running."
Write-Host "Rate limit: 3 requests per minute per IP (shared across all servers)"
Write-Host ""
Write-Host "Servers:"
Write-Host "  http://localhost:8001/api"
Write-Host "  http://localhost:8002/api"
Write-Host "  http://localhost:8003/api"
Write-Host ""
Write-Host "Manual test (copy and paste these):"
Write-Host "  curl http://localhost:8001/api   -> OK - served by :8001"
Write-Host "  curl http://localhost:8002/api   -> OK - served by :8002"
Write-Host "  curl http://localhost:8003/api   -> OK - served by :8003"
Write-Host "  (4th request to any server)      -> 429 Too Many Requests"
Write-Host ""
Write-Host "Press Ctrl+C to stop all servers."

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