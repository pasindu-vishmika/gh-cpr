<#
.SYNOPSIS
    Build and install (or update) the `cpr` CLI on Windows.

.DESCRIPTION
    Re-run this script any time to update to the latest version: it pulls the
    newest source (when run from a git clone), rebuilds, and replaces the old
    binary in place.

    Prerequisites checked automatically: Go, GitHub CLI (gh), and that gh is
    authenticated.

    Install location defaults to %LOCALAPPDATA%\Programs\cpr. Override with:
        $env:CPR_INSTALL_DIR = "C:\tools\cpr"; .\install.ps1
#>

#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$BinName    = 'cpr.exe'
$InstallDir = if ($env:CPR_INSTALL_DIR) { $env:CPR_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\cpr' }

function Info($m) { Write-Host "==> $m" -ForegroundColor Cyan }
function Ok($m)   { Write-Host " [ok] $m" -ForegroundColor Green }
function Warn($m) { Write-Host " [!] $m"  -ForegroundColor Yellow }
function Fail($m) { Write-Host " [x] $m"  -ForegroundColor Red; exit 1 }

# ── 1. Prerequisites ──────────────────────────────────────────────────────────
Info "Checking prerequisites"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Fail "Go is not installed. Install it from https://go.dev/dl/ and try again."
}
Ok "Go found ($((go version)))"

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    Fail "GitHub CLI (gh) is not installed. Install it from https://cli.github.com/ and try again."
}
Ok "GitHub CLI found"

gh auth status *> $null
if ($LASTEXITCODE -ne 0) {
    Fail "GitHub CLI is not authenticated. Run: gh auth login"
}
Ok "GitHub CLI authenticated"

# ── 2. Move into the repo (this script's directory) ─────────────────────────────
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# ── 3. Pull the latest source for monthly updates (only if a git clone) ─────────
if ((Test-Path .git) -and (Get-Command git -ErrorAction SilentlyContinue)) {
    Info "Updating source (git pull --ff-only)"
    git pull --ff-only *> $null
    if ($LASTEXITCODE -eq 0) { Ok "Source is up to date" }
    else { Warn "Could not fast-forward (local changes or detached HEAD); building current source" }
}

# ── 4. Build ────────────────────────────────────────────────────────────────────
Info "Building $BinName"
go mod tidy
go build -o $BinName .
Ok "Built .\$BinName"

# ── 5. Install (remove old, copy new) ───────────────────────────────────────────
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
$Dest = Join-Path $InstallDir $BinName
if (Test-Path $Dest) { Remove-Item $Dest -Force }   # remove any previous version
Move-Item $BinName $Dest -Force
Ok "Installed $BinName -> $Dest"

# ── 6. PATH (persist permanently to the user environment) ───────────────────────
# SetEnvironmentVariable at 'User' scope writes to the registry, so the change
# survives reboots and new terminals (not just this session).
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$entries  = @()
if ($userPath) { $entries = $userPath.Split(';') | Where-Object { $_ -ne '' } }

if ($entries -notcontains $InstallDir) {
    $newUserPath = (@($entries + $InstallDir) -join ';')
    [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
    Ok "Added $InstallDir to your user PATH (permanent)."
} else {
    Ok "$InstallDir is already on your user PATH (permanent)."
}

# Make it usable right now, without waiting for a terminal restart.
if (($env:Path -split ';') -notcontains $InstallDir) {
    $env:Path = "$env:Path;$InstallDir"
}

Write-Host ""
Ok "Done. 'cpr' works in this session and in every new terminal."
