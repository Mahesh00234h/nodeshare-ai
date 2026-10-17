# PowerShell Windows Service Installer for Distributed AI Cluster Node Agent
param (
    [string]$ControllerURL = "https://llm-cluster.onrender.com:443",
    [string]$EnrollmentToken = "",
    [string]$InstallDir = "C:\Program Files\DistributedAI\NodeAgent"
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "Continue"

Write-Host "=== Distributed AI Node Agent Installer ===" -ForegroundColor Cyan

# 1. Create Directories
$DataDir = "C:\ProgramData\DistributedAI"
$BinDir = "$DataDir\bin"
$ModelDir = "$DataDir\models"
$ModelPath = "$ModelDir\qwen2.5-0.5b-instruct-q4_k_m.gguf"
$LlamaCli = "$BinDir\llama-cli.exe"
$LlamaRpc = "$BinDir\llama-rpc-server.exe"
$ModelUrl = "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"
$LlamaReleaseApi = "https://api.github.com/repos/ggml-org/llama.cpp/releases?per_page=20"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path "$DataDir\config" | Out-Null
New-Item -ItemType Directory -Force -Path "$DataDir\identity" | Out-Null
New-Item -ItemType Directory -Force -Path "$DataDir\logs" | Out-Null
New-Item -ItemType Directory -Force -Path $ModelDir | Out-Null
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

function Invoke-DownloadWithRetry {
    param ([string]$Uri, [string]$Destination)
    $partial = "$Destination.part"
    for ($attempt = 1; $attempt -le 5; $attempt++) {
        try {
            Invoke-WebRequest -Uri $Uri -OutFile $partial -UserAgent "Mozilla/5.0" -TimeoutSec 1800
            Move-Item -Path $partial -Destination $Destination -Force
            return
        } catch {
            Remove-Item -Path $partial -Force -ErrorAction SilentlyContinue
            if ($attempt -eq 5) { throw }
            Start-Sleep -Seconds ($attempt * $attempt)
        }
    }
}

$hasRuntimeDll = @(Get-ChildItem -Path $BinDir -Filter *.dll -File -ErrorAction SilentlyContinue).Count -gt 0
if (-not (Test-Path $LlamaCli) -or -not $hasRuntimeDll) {
    $llamaZip = Join-Path $env:TEMP "distributed-ai-llama-cuda.zip"
    Write-Host "Downloading llama.cpp runtime binaries..." -ForegroundColor Yellow
    $releases = Invoke-RestMethod -Uri $LlamaReleaseApi -Headers @{ "User-Agent" = "DistributedAI-NodeAgent-Installer" }
    $asset = $releases.assets | Where-Object { $_.name -match '(?i)^llama-.*win.*cuda.*x64.*\.zip$' } | Select-Object -First 1
    if (-not $asset) {
        $asset = $releases.assets | Where-Object { $_.name -match '(?i)^llama-.*win.*cpu.*x64.*\.zip$' } | Select-Object -First 1
    }
    if (-not $asset) { throw "No Windows llama.cpp release asset found." }
    Invoke-DownloadWithRetry -Uri $asset.browser_download_url -Destination $llamaZip
    Expand-Archive -Path $llamaZip -DestinationPath $BinDir -Force
    Remove-Item -Path $llamaZip -Force -ErrorAction SilentlyContinue
}
if (-not (Test-Path $LlamaCli)) {
    Write-Warning "llama.cpp CLI is unavailable. The node service will still start, but inference waits until llama.cpp is installed."
} elseif (-not (Test-Path $LlamaRpc)) {
    Write-Warning "llama-rpc-server.exe is unavailable. Single-node inference remains available; distributed RPC worker mode is disabled."
}

if (-not (Test-Path $ModelPath)) {
    Write-Host "Downloading GGUF model..." -ForegroundColor Yellow
    Invoke-DownloadWithRetry -Uri $ModelUrl -Destination $ModelPath
}

# 2. Copy Executable
if (Test-Path "node-agent.exe") {
    Copy-Item "node-agent.exe" -Destination "$InstallDir\node-agent.exe" -Force
} else {
    Write-Host "Warning: node-agent.exe not found in current directory. Build binary first with 'go build ./cmd/node-agent'." -ForegroundColor Yellow
}

# 3. Write Config
$ConfigObj = @{
    controller_url = $ControllerURL
    enrollment_token = $EnrollmentToken
    node_data_dir = $DataDir
    log_level = "INFO"
    telemetry_interval_seconds = 5
    heartbeat_interval_seconds = 5
    max_concurrent_tasks = 4
}
$ConfigJson = $ConfigObj | ConvertTo-Json -Depth 4
Set-Content -Path "$DataDir\config\config.json" -Value $ConfigJson

# 4. Install & Enable Windows Service
$ServiceName = "DistributedAINodeAgent"
$BinaryPath = "`"$InstallDir\node-agent.exe`" --config `"$DataDir\config\config.json`""

$ExistingSvc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($ExistingSvc) {
    Write-Host "Stopping existing service..." -ForegroundColor Yellow
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

Write-Host "Registering Windows Service '$ServiceName'..." -ForegroundColor Green
sc.exe create $ServiceName binPath= "$BinaryPath" start= delayed-auto DisplayName= "Distributed AI Compute Node Agent" | Out-Null
sc.exe config $ServiceName depend= Tcpip | Out-Null
sc.exe description $ServiceName "Background compute worker node agent for private distributed LLM cluster." | Out-Null
sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/20000 | Out-Null
sc.exe failureflag $ServiceName 1 | Out-Null

# 5. Start Service
Write-Host "Starting Windows Service..." -ForegroundColor Green
Start-Service -Name $ServiceName

Write-Host "=== Installation Complete! Node Agent is running as a Windows Service. ===" -ForegroundColor Green
