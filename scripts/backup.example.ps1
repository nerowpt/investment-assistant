# MVP-1 manual backup (before `inv backup` exists).
# Usage (PowerShell, from repo root):
#   .\scripts\backup.example.ps1 -AccountId default -Mode lite
#
# Requires: DATA_ROOT env or defaults to .\data

param(
    [string]$AccountId = "default",
    [ValidateSet("lite", "full")]
    [string]$Mode = "lite",
    [string]$DataRoot = "",
    [string]$BackupRoot = ""
)

$ErrorActionPreference = "Stop"

if ($env:IA_ACCOUNT_ID) { $AccountId = $env:IA_ACCOUNT_ID }
if (-not $DataRoot) {
    if ($env:DATA_ROOT) { $DataRoot = $env:DATA_ROOT }
    else { $DataRoot = Join-Path $PSScriptRoot ".." "data" }
}
$DataRoot = (Resolve-Path $DataRoot -ErrorAction SilentlyContinue).Path
if (-not $DataRoot) { $DataRoot = (New-Item -ItemType Directory -Force -Path (Join-Path $PSScriptRoot ".." "data")).FullName }

if (-not $BackupRoot) {
    if ($env:BACKUP_ROOT) { $BackupRoot = $env:BACKUP_ROOT }
    else { $BackupRoot = Join-Path $DataRoot "_backups" }
}

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$dest = Join-Path $BackupRoot $AccountId $timestamp
$src = Join-Path $DataRoot "accounts" $AccountId

if (-not (Test-Path $src)) {
    Write-Error "Account path not found: $src"
}

New-Item -ItemType Directory -Force -Path $dest | Out-Null

function Copy-Tree($rel) {
    $from = Join-Path $src $rel
    if (Test-Path $from) {
        Copy-Item -Path $from -Destination (Join-Path $dest $rel) -Recurse -Force
    }
}

Copy-Tree "state"
Copy-Tree "db"
if ($Mode -eq "full") {
    Copy-Tree "library"
    Copy-Tree "reports"
}

$manifest = @{
    backup_id   = $timestamp
    account_id  = $AccountId
    mode        = $Mode
    created_at  = (Get-Date).ToString("o")
    data_root   = $DataRoot
    script      = "backup.example.ps1"
} | ConvertTo-Json -Depth 4

$manifest | Set-Content -Path (Join-Path $dest "manifest.json") -Encoding UTF8

Write-Host "Backup created: $dest"
Write-Host "Next: run doctor after any restore; prune old backups manually or via inv backup prune (stage 4)."
