# fpm Installer for Windows
# Usage: irm https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.ps1 | iex
#
# Options (set before running):
#   $env:FPM_INSTALL_DIR = "C:\custom\path"

$ErrorActionPreference = "Stop"

$Repo = "Kartikey2011yadav/fpm"
$BinaryName = "fpm.exe"

# ─────────────────────────────────────────────────────────────────────────────
# Platform Detection
# ─────────────────────────────────────────────────────────────────────────────

$Arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Error "32-bit systems are not supported."
    exit 1
}

$Platform = "windows-$Arch"

Write-Host ""
Write-Host "  ███████╗██████╗ ███╗   ███╗" -ForegroundColor Cyan
Write-Host "  ██╔════╝██╔══██╗████╗ ████║" -ForegroundColor Cyan
Write-Host "  █████╗  ██████╔╝██╔████╔██║" -ForegroundColor Cyan
Write-Host "  ██╔══╝  ██╔═══╝ ██║╚██╔╝██║" -ForegroundColor Cyan
Write-Host "  ██║     ██║     ██║ ╚═╝ ██║" -ForegroundColor Cyan
Write-Host "  ╚═╝     ╚═╝     ╚═╝     ╚═╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Fast Package Manager for Python" -ForegroundColor White
Write-Host "  Platform: $Platform" -ForegroundColor DarkGray
Write-Host ""

# ─────────────────────────────────────────────────────────────────────────────
# Installation Directory
# ─────────────────────────────────────────────────────────────────────────────

$InstallDir = if ($env:FPM_INSTALL_DIR) {
    $env:FPM_INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA "fpm\bin"
}

Write-Host "  Installing to: $InstallDir" -ForegroundColor White

# ─────────────────────────────────────────────────────────────────────────────
# Get Latest Release
# ─────────────────────────────────────────────────────────────────────────────

Write-Host "  Checking latest version..." -ForegroundColor Blue

try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "fpm-installer" }
    $Version = $Release.tag_name -replace "^v", ""
} catch {
    Write-Error "Failed to fetch latest release: $_"
    exit 1
}

Write-Host "  Latest version: $Version" -ForegroundColor Green

# ─────────────────────────────────────────────────────────────────────────────
# Download Binary
# ─────────────────────────────────────────────────────────────────────────────

$AssetName = "fpm-v$Version-windows-$Arch.exe"
$DownloadUrl = "https://github.com/$Repo/releases/download/v$Version/$AssetName"
$FallbackUrl = "https://github.com/$Repo/releases/download/v$Version/fpm-$Version-windows-$Arch.exe"

$TempFile = Join-Path $env:TEMP "fpm-install-$([System.IO.Path]::GetRandomFileName()).exe"

Write-Host "  Downloading fpm $Version..." -ForegroundColor Blue

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempFile -UseBasicParsing
} catch {
    try {
        Invoke-WebRequest -Uri $FallbackUrl -OutFile $TempFile -UseBasicParsing
    } catch {
        Write-Error "Download failed. Check your network connection."
        exit 1
    }
}

$FileSize = [math]::Round((Get-Item $TempFile).Length / 1MB, 1)
Write-Host "  Downloaded ($FileSize MB)" -ForegroundColor Green

# ─────────────────────────────────────────────────────────────────────────────
# Verify Integrity
# ─────────────────────────────────────────────────────────────────────────────

Write-Host "  Verifying integrity..." -ForegroundColor Blue

$ChecksumUrl = "https://github.com/$Repo/releases/download/v$Version/checksums.txt"
try {
    $Checksums = (Invoke-WebRequest -Uri $ChecksumUrl -UseBasicParsing).Content
    $ExpectedLine = ($Checksums -split "`n") | Where-Object { $_ -match "windows-$Arch" } | Select-Object -First 1
    if ($ExpectedLine) {
        $ExpectedHash = ($ExpectedLine -split "\s+")[0]
        $ActualHash = (Get-FileHash -Path $TempFile -Algorithm SHA256).Hash.ToLower()
        if ($ActualHash -ne $ExpectedHash) {
            Remove-Item $TempFile -Force
            Write-Error "Checksum mismatch! Expected: $ExpectedHash, Got: $ActualHash"
            exit 1
        }
        Write-Host "  Integrity verified (SHA256)" -ForegroundColor Green
    } else {
        Write-Host "  No checksum found for this platform (skipping)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  Could not download checksums (skipping verification)" -ForegroundColor Yellow
}

# ─────────────────────────────────────────────────────────────────────────────
# Install
# ─────────────────────────────────────────────────────────────────────────────

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$DestPath = Join-Path $InstallDir $BinaryName
Move-Item -Path $TempFile -Destination $DestPath -Force

Write-Host "  Installed to $DestPath" -ForegroundColor Green

# ─────────────────────────────────────────────────────────────────────────────
# Add to PATH
# ─────────────────────────────────────────────────────────────────────────────

$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$InstallDir;$UserPath", "User")
    $env:PATH = "$InstallDir;$env:PATH"
    Write-Host "  Added to PATH (user environment)" -ForegroundColor Green
} else {
    Write-Host "  Already in PATH" -ForegroundColor Green
}

# ─────────────────────────────────────────────────────────────────────────────
# Done
# ─────────────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "  Get started:" -ForegroundColor White
Write-Host "    fpm init myproject" -ForegroundColor DarkGray
Write-Host "    cd myproject" -ForegroundColor DarkGray
Write-Host "    fpm install requests" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  Run 'fpm --help' for all commands." -ForegroundColor DarkGray
Write-Host "  Restart your terminal if 'fpm' is not found." -ForegroundColor Yellow
Write-Host ""
