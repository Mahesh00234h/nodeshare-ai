# Setup and Installer Script for Distributed AI Node Agent & Llama Runtime
# Usage: powershell -ExecutionPolicy Bypass -File setup_node.ps1 [-Role primary|worker] [-ControllerUrl https://llm-cluster.onrender.com:443]

param (
    [string]$Role = "primary", # "primary" or "worker"
    [string]$ControllerUrl = "https://llm-cluster.onrender.com:443",
    [string]$EnrollmentToken = "dev-enrollment-token",
    [string]$RPCPort = "50052",
    [switch]$SkipModelDownload = $false,
    [switch]$Daemon = $false
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "Continue"

Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host "   Distributed AI System - Automated Physical PC Node Installer   " -ForegroundColor Cyan
Write-Host "=================================================================" -ForegroundColor Cyan

# 1. Create System Directories
$BaseDir   = "C:\ProgramData\DistributedAI"
$BinDir    = "$BaseDir\bin"
$ModelDir  = "$BaseDir\models"
$LogDir    = "$BaseDir\logs"
$ConfigDir = "$BaseDir\configs"

Write-Host "`n[1/5] Creating system directories in $BaseDir..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
New-Item -ItemType Directory -Force -Path $ModelDir | Out-Null
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
Write-Host "  [OK] Directories ready." -ForegroundColor Green

# 2. Check / Copy Node Agent Binaries
Write-Host "`n[2/5] Deploying Node Agent binaries..." -ForegroundColor Yellow
$LocalAgentBin = Join-Path $PSScriptRoot "..\bin\node-agent.exe"
$TargetAgentBin = "$BinDir\node-agent.exe"

if (Test-Path $LocalAgentBin) {
    Copy-Item -Path $LocalAgentBin -Destination $TargetAgentBin -Force
    Write-Host "  [OK] Copied node-agent.exe to $TargetAgentBin" -ForegroundColor Green
} else {
    Write-Host "  [i] node-agent.exe not found locally, checking target $TargetAgentBin..." -ForegroundColor Cyan
    if (-not (Test-Path $TargetAgentBin)) {
        Write-Warning "node-agent.exe is missing! Please build node-agent first using 'go build -o bin/node-agent.exe ./cmd/node-agent'."
    }
}

# 3. Download / Verify Llama.cpp CUDA Binaries
Write-Host "`n[3/5] Verifying llama.cpp CUDA binaries (llama-cli.exe and llama-rpc-server.exe)..." -ForegroundColor Yellow
$LlamaCli = "$BinDir\llama-cli.exe"
$LlamaRpc = "$BinDir\llama-rpc-server.exe"

if (-not (Test-Path $LlamaCli)) {
    Write-Host "  [i] llama.cpp binaries not detected in $BinDir. Downloading CUDA release package..." -ForegroundColor Cyan
    $ReleaseApi = "https://api.github.com/repos/ggml-org/llama.cpp/releases?per_page=20"
    $ZipPath = "$env:TEMP\llama-cuda.zip"
    
    $releases = Invoke-RestMethod -Uri $ReleaseApi -Headers @{ "User-Agent" = "DistributedAI-NodeAgent-Installer" }
    $asset = $releases.assets | Where-Object { $_.name -match '(?i)^llama-.*win.*cuda.*x64.*\.zip$' } | Select-Object -First 1
    if (-not $asset) { $asset = $releases.assets | Where-Object { $_.name -match '(?i)^llama-.*win.*cpu.*x64.*\.zip$' } | Select-Object -First 1 }
    if (-not $asset) { throw "No Windows llama.cpp release asset found." }
    Write-Host "  Downloading $($asset.name) ..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $ZipPath -UserAgent "Mozilla/5.0"
    Write-Host "  Extracting CUDA binaries to $BinDir..." -ForegroundColor Cyan
    Expand-Archive -Path $ZipPath -DestinationPath $BinDir -Force
    Remove-Item -Path $ZipPath -Force
    Write-Host "  [OK] Successfully installed llama.cpp CUDA binaries!" -ForegroundColor Green
} else {
    Write-Host "  [OK] llama.cpp binaries present in $BinDir." -ForegroundColor Green
}

# 4. Download GGUF Model
if (-not $SkipModelDownload) {
    Write-Host "`n[4/5] Checking GGUF LLM Model..." -ForegroundColor Yellow
    $ModelPath = "$ModelDir\qwen2.5-0.5b-instruct-q4_k_m.gguf"
    if (-not (Test-Path $ModelPath)) {
        Write-Host "  [i] Downloading lightweight Qwen2.5 0.5B GGUF test model..." -ForegroundColor Cyan
        $ModelUrl = "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"
        $downloaded = $false
        for ($attempt = 1; $attempt -le 5 -and -not $downloaded; $attempt++) {
            try {
                $partPath = "$ModelPath.part"
                Invoke-WebRequest -Uri $ModelUrl -OutFile $partPath -UserAgent "Mozilla/5.0" -TimeoutSec 1800
                Move-Item -Path $partPath -Destination $ModelPath -Force
                $downloaded = $true
            } catch {
                Remove-Item -Path "$ModelPath.part" -Force -ErrorAction SilentlyContinue
                if ($attempt -lt 5) {
                    Start-Sleep -Seconds ($attempt * $attempt)
                }
            }
        }
        if (-not $downloaded) {
            Write-Warning "Model download failed after 5 attempts. The node agent will retry automatically at startup and before inference."
        } else {
            Write-Host "  [OK] Downloaded model to $ModelPath" -ForegroundColor Green
        }
    } else {
        Write-Host "  [OK] GGUF model present at $ModelPath" -ForegroundColor Green
    }
} else {
    Write-Host "`n[4/5] Skipping model download (Role: $Role)." -ForegroundColor Gray
}

# 5. Launch RPC Worker & Node Agent
Write-Host "`n[5/5] Launching Distributed Node Services..." -ForegroundColor Yellow

if (Test-Path $LlamaRpc) {
    Write-Host "  [i] Starting llama-rpc-server on port $RPCPort in background..." -ForegroundColor Cyan
    Start-Process -FilePath $LlamaRpc -ArgumentList "-H 0.0.0.0 -p $RPCPort" -WindowStyle Hidden
    Write-Host "  [OK] RPC Worker listening on 0.0.0.0:$RPCPort" -ForegroundColor Green
}

# 6. Configure Windows Auto-Start on Boot
Write-Host "`n[6/6] Configuring Windows Auto-Start on Boot..." -ForegroundColor Yellow
$ServiceName = "DistributedAINodeAgent"
$BinaryPath = "`"$TargetAgentBin`" --controller-url `"$ControllerUrl`" --enrollment-token `"$EnrollmentToken`""
$existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existing) {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}
sc.exe create $ServiceName binPath= "$BinaryPath" start= delayed-auto DisplayName= "Distributed AI Compute Node Agent" | Out-Null
sc.exe config $ServiceName depend= Tcpip | Out-Null
sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/20000 | Out-Null
sc.exe failureflag $ServiceName 1 | Out-Null
Start-Service -Name $ServiceName
Write-Host "  [OK] Registered Windows service $ServiceName for automatic startup" -ForegroundColor Green

Write-Host "`nStarting Distributed Node Agent..." -ForegroundColor Green
Write-Host "Connecting to Controller: $ControllerUrl" -ForegroundColor Cyan

if (Test-Path $TargetAgentBin) {
    if ($Daemon) {
        Write-Host "  [i] The Windows service owns the Node Agent and will restart it after reboot or failure." -ForegroundColor Cyan
        Write-Host "  [OK] Node Agent service is running silently in the background! You can close PowerShell now." -ForegroundColor Green
    }
    if (-not $Daemon) {
        Start-Process -FilePath $TargetAgentBin -ArgumentList "--foreground", "--controller-url", $ControllerUrl, "--enrollment-token", $EnrollmentToken -Wait -NoNewWindow
    }
}
if (-not (Test-Path $TargetAgentBin)) {
    Write-Error ('Cannot start Node Agent binary: ' + $TargetAgentBin + ' not found.')
}
