# Box CLI Installation Script for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/gravelight-studio/box/main/install.ps1 | iex

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\box"
)

$ErrorActionPreference = "Stop"

# Colors for output
function Write-Info {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

Write-Host "🎯 Box CLI Installer for Windows" -ForegroundColor Cyan
Write-Host ""

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    if ([Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE") -eq "ARM64") {
        "arm64"
    } else {
        "amd64"
    }
} else {
    Write-Error "32-bit Windows is not supported"
    exit 1
}

Write-Host "📦 Detected platform: windows-$Arch"
Write-Host ""

# Get latest version if not specified
if ($Version -eq "latest") {
    Write-Host "🔍 Fetching latest version..."
    try {
        $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/gravelight-studio/box/releases/latest"
        $Version = $Release.tag_name -replace '^v', ''
    } catch {
        Write-Error "Failed to fetch latest version: $_"
        exit 1
    }
}

Write-Host "📥 Downloading Box CLI v$Version..."
$BinaryName = "box-windows-$Arch.exe"
$DownloadUrl = "https://github.com/gravelight-studio/box/releases/download/v$Version/$BinaryName"
$ChecksumUrl = "https://github.com/gravelight-studio/box/releases/download/v$Version/$BinaryName.sha256"

# Create temp directory
$TempDir = Join-Path $env:TEMP "box-install-$(Get-Random)"
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    # Download binary
    $BinaryPath = Join-Path $TempDir $BinaryName
    Write-Host "Downloading from $DownloadUrl..."
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $BinaryPath -UseBasicParsing
    } catch {
        Write-Error "Failed to download binary: $_"
        exit 1
    }

    # Download and verify checksum
    $ChecksumPath = Join-Path $TempDir "$BinaryName.sha256"
    try {
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile $ChecksumPath -UseBasicParsing

        Write-Host "🔐 Verifying checksum..."
        $ExpectedHash = (Get-Content $ChecksumPath).Split()[0].ToUpper()
        $ActualHash = (Get-FileHash -Path $BinaryPath -Algorithm SHA256).Hash.ToUpper()

        if ($ExpectedHash -ne $ActualHash) {
            Write-Error "Checksum verification failed!"
            Write-Host "Expected: $ExpectedHash"
            Write-Host "Actual:   $ActualHash"
            exit 1
        }
    } catch {
        Write-Warning "Failed to download/verify checksum, skipping verification"
    }

    # Create install directory
    Write-Host "📦 Installing to $InstallDir..."
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

    # Install binary
    $DestPath = Join-Path $InstallDir "box.exe"
    Copy-Item -Path $BinaryPath -Destination $DestPath -Force

    Write-Host ""
    Write-Info "Box CLI installed successfully!"
    Write-Host ""
    Write-Host "📍 Location: $DestPath"
    Write-Host "📦 Version: v$Version"
    Write-Host ""

    # Check if directory is in PATH
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        Write-Warning "$InstallDir is not in your PATH"
        Write-Host ""
        Write-Host "Would you like to add it to your PATH? (Y/N)" -ForegroundColor Yellow
        $Response = Read-Host

        if ($Response -eq "Y" -or $Response -eq "y") {
            $NewPath = "$UserPath;$InstallDir"
            [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
            Write-Info "Added to PATH! Restart your terminal for changes to take effect."
        } else {
            Write-Host ""
            Write-Host "To add to PATH manually, run:" -ForegroundColor Yellow
            Write-Host '  $env:Path += ";' + $InstallDir + '"' -ForegroundColor Cyan
            Write-Host "Or add it permanently in System Settings > Environment Variables"
        }
    }

    # Test installation
    Write-Host ""
    Write-Host "🎉 Installation complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Get started:" -ForegroundColor Cyan
    Write-Host "  box init my-app          # Create a new project"
    Write-Host "  box build --help         # Build deployment artifacts"
    Write-Host ""

    if ($UserPath -like "*$InstallDir*") {
        & $DestPath version
    } else {
        Write-Host "Run 'box version' after adding to PATH or restart your terminal" -ForegroundColor Yellow
    }

} finally {
    # Cleanup
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
