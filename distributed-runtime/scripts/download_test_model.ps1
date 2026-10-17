# PowerShell script to download a small GGUF model for testing distributed inference
param (
    [string]$TargetDir = "C:\ProgramData\DistributedAI\models"
)

Write-Host "Creating target directory: $TargetDir"
New-Item -ItemType Directory -Force -Path $TargetDir | Out-Null

$ModelUrl = "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"
$OutputFile = Join-Path $TargetDir "qwen2.5-0.5b-instruct-q4_k_m.gguf"

if (Test-Path $OutputFile) {
    Write-Host "Model file already exists at: $OutputFile"
} else {
    Write-Host "Downloading test GGUF model from $ModelUrl..."
    Invoke-WebRequest -Uri $ModelUrl -OutFile $OutputFile -UseBasicParsing
    Write-Host "Download complete: $OutputFile"
}
