# easyrice installer (Windows PowerShell)
#
# Environment variables:
#   EASYRICE_INSTALL_DIR      Install directory (default: $env:LOCALAPPDATA\Programs\easyrice)
#   EASYRICE_VERSION          Version to install (default: latest)
#   EASYRICE_RELEASE_BASE_URL Base URL for release assets
#                             (default: https://github.com/guneet-xyz/easyrice/releases/download)
#                             Override for mirrors or local testing.
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Repo            = "guneet-xyz/easyrice"
$Binary          = "easyrice"
$Symlink         = "rice"
$InstallDir      = if ($env:EASYRICE_INSTALL_DIR) { $env:EASYRICE_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\easyrice" }
$Version         = if ($env:EASYRICE_VERSION) { $env:EASYRICE_VERSION } else { "latest" }
$ReleaseBaseUrl  = if ($env:EASYRICE_RELEASE_BASE_URL) { $env:EASYRICE_RELEASE_BASE_URL } else { "https://github.com/$Repo/releases/download" }

function Write-Info($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Warn($msg) { Write-Host "warn: $msg" -ForegroundColor Yellow }
function Write-Err($msg)  { Write-Host "error: $msg" -ForegroundColor Red; exit 1 }

function Read-Yes($prompt) {
    $answer = Read-Host -Prompt $prompt
    return ($answer -match '^(y|yes)$')
}

# Architecture detection
switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { $Arch = 'amd64' }
    'x64'   { $Arch = 'amd64' }
    'ARM64' { $Arch = 'arm64' }
    default { Write-Err "Unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

# Resolve version
if ($Version -eq 'latest') {
    Write-Info "Resolving latest version..."
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        $Version = $release.tag_name
    } catch {
        Write-Err "Could not determine latest version: $_"
    }
    if (-not $Version) { Write-Err "Could not determine latest version" }
}

# Normalize: prepend 'v' if missing
if ($Version -notmatch '^v') {
    $Version = "v$Version"
}

$BinaryExe  = "$Binary.exe"
$SymlinkExe = "$Symlink.exe"
$InstalledBinary = Join-Path $InstallDir $BinaryExe

# Detect existing install
if (Test-Path $InstalledBinary) {
    try {
        $current = & $InstalledBinary version 2>$null
    } catch {
        $current = "unknown"
    }
    Write-Host "$Binary is already installed ($current)." -ForegroundColor Yellow
    if ([Environment]::UserInteractive) {
        if (-not (Read-Yes "Reinstall $Version? [y/N]")) {
            Write-Info "Cancelled."
            exit 0
        }
    }
}

$Asset    = "easyrice-$Version-windows-$Arch.exe"
$Url      = "$ReleaseBaseUrl/$Version/$Asset"
$SumsUrl  = "$ReleaseBaseUrl/$Version/checksums.txt"

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("easyrice-" + [System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null

try {
    $TmpFile  = Join-Path $TmpDir $Asset
    $TmpSums  = Join-Path $TmpDir 'checksums.txt'

    Write-Info "Downloading $Asset..."
    try {
        Invoke-WebRequest -Uri $Url -OutFile $TmpFile -UseBasicParsing
    } catch {
        Write-Err "Failed to download $Url (no prebuilt binary for windows/$Arch in $Version?)"
    }

    Write-Info "Downloading checksums..."
    $haveSums = $true
    try {
        Invoke-WebRequest -Uri $SumsUrl -OutFile $TmpSums -UseBasicParsing
    } catch {
        Write-Warn "Could not fetch checksums.txt; skipping verification"
        $haveSums = $false
    }

    if ($haveSums) {
        Write-Info "Verifying checksum..."
        $expected = $null
        foreach ($line in Get-Content $TmpSums) {
            if ($line -match "\s$([regex]::Escape($Asset))$") {
                $expected = ($line -split '\s+')[0].ToLower()
                break
            }
        }
        if (-not $expected) { Write-Err "Checksum for $Asset not found in checksums.txt" }
        $actual = (Get-FileHash -Algorithm SHA256 $TmpFile).Hash.ToLower()
        if ($expected -ne $actual) {
            Write-Err "Checksum mismatch! expected=$expected actual=$actual"
        }
        Write-Info "Checksum OK"
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Move-Item -Force -Path $TmpFile -Destination $InstalledBinary
    Write-Info "Installed $Binary $Version to $InstalledBinary"

    # Symlink (Developer Mode required) with .cmd shim fallback
    $SymlinkPath = Join-Path $InstallDir $SymlinkExe
    $ShimPath    = Join-Path $InstallDir "$Symlink.cmd"
    try {
        if (Test-Path $SymlinkPath) { Remove-Item -Force $SymlinkPath }
        New-Item -ItemType SymbolicLink -Path $SymlinkPath -Target $InstalledBinary -Force -ErrorAction Stop | Out-Null
        if (Test-Path $ShimPath) { Remove-Item -Force $ShimPath }
        Write-Info "Created symlink $SymlinkExe -> $BinaryExe"
    } catch {
        Set-Content -Path $ShimPath -Value "@`"%~dp0$BinaryExe`" %*"
        Write-Warn "Could not create symlink (requires Developer Mode). Created $Symlink.cmd shim instead."
    }
} finally {
    if (Test-Path $TmpDir) { Remove-Item -Recurse -Force $TmpDir }
}

# PATH check (User scope only)
$userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
if (-not $userPath) { $userPath = '' }
$pathEntries = $userPath -split ';' | Where-Object { $_ -ne '' }
$onPath = $pathEntries | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') }

if ($onPath) {
    Write-Info "$InstallDir is already in your PATH. You're all set."
    exit 0
}

Write-Host ""
Write-Host "$InstallDir is not in your PATH." -ForegroundColor Yellow
if (-not [Environment]::UserInteractive) {
    Write-Host "Add it manually:`n  [Environment]::SetEnvironmentVariable('PATH', `"$userPath;$InstallDir`", 'User')"
    exit 0
}

if (Read-Yes "Add it to your User PATH? [y/N]") {
    $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
    Write-Info "Added to User PATH. Open a new shell for the change to take effect."
} else {
    Write-Host "Add this to your User PATH manually:`n  $InstallDir"
}
