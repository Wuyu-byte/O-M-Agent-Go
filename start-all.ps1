param(
    [int]$BackendPort = 6872,
    [int]$FrontendPort = 8080
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Require-Command {
    param(
        [string]$Name,
        [string]$Hint
    )

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Command not found: $Name. $Hint"
    }
}

function Get-PythonCommand {
    if (Get-Command py -ErrorAction SilentlyContinue) {
        return "py -3"
    }
    if (Get-Command python -ErrorAction SilentlyContinue) {
        return "python"
    }
    if (Get-Command python3 -ErrorAction SilentlyContinue) {
        return "python3"
    }
    throw "Python was not found. Please install Python 3, or start the frontend with another static file server."
}

function Warn-IfPortBusy {
    param(
        [int]$Port,
        [string]$Name
    )

    $busy = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue |
        Where-Object { $_.State -in @("Listen", "Established") } |
        Select-Object -First 1

    if ($busy) {
        Write-Host "Notice: $Name port $Port may already be in use. Check the service window logs." -ForegroundColor Yellow
    }
}

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$DockerDir = Join-Path $Root "manifest\docker"
$ComposeFile = Join-Path $DockerDir "docker-compose.yml"
$ConfigDir = Join-Path $Root "manifest\config"
$FrontendDir = Join-Path $Root "Frontend"

Write-Host "Agent launcher" -ForegroundColor Green
Write-Host "Project root: $Root"

Require-Command docker "Please install and start Docker Desktop first."
Require-Command go "Please install Go and make sure the go command is in PATH."
$PythonCommand = Get-PythonCommand

if (-not (Test-Path $ComposeFile)) {
    throw "Docker Compose file not found: $ComposeFile"
}
$ConfigFile = Join-Path $ConfigDir "config.yaml"
$FrontendIndex = Join-Path $FrontendDir "index.html"

if (-not (Test-Path $ConfigFile)) {
    throw "Backend config file not found: $ConfigFile"
}
if (-not (Test-Path $FrontendIndex)) {
    throw "Frontend entry file not found: $FrontendIndex"
}

Warn-IfPortBusy -Port $BackendPort -Name "backend"
Warn-IfPortBusy -Port $FrontendPort -Name "frontend"

Write-Step "Starting Docker data services: Milvus / Etcd / MinIO / Attu"
Push-Location $DockerDir
try {
    docker compose up -d
}
finally {
    Pop-Location
}

Write-Step "Starting Go backend: http://localhost:$BackendPort"
$BackendCommand = @"
`$env:GF_GCFG_PATH = '$ConfigDir'
`$env:GF_GCFG_FILE = 'config.yaml'
Set-Location '$Root'
go run .
"@
Start-Process powershell.exe -WindowStyle Normal -ArgumentList @(
    "-NoExit",
    "-ExecutionPolicy", "Bypass",
    "-Command", $BackendCommand
)

Write-Step "Starting frontend static server: http://localhost:$FrontendPort"
$FrontendCommand = @"
Set-Location '$FrontendDir'
$PythonCommand -m http.server $FrontendPort
"@
Start-Process powershell.exe -WindowStyle Normal -ArgumentList @(
    "-NoExit",
    "-ExecutionPolicy", "Bypass",
    "-Command", $FrontendCommand
)

Write-Host ""
Write-Host "Startup commands have been sent:" -ForegroundColor Green
Write-Host "  Frontend: http://localhost:$FrontendPort"
Write-Host "  Backend: http://localhost:$BackendPort/api"
Write-Host "  Attu: http://localhost:8000"
Write-Host ""
Write-Host "To stop services:" -ForegroundColor Yellow
Write-Host "  1. Press Ctrl+C in the backend/frontend windows"
Write-Host "  2. Stop data services with: cd manifest\docker; docker compose down"
