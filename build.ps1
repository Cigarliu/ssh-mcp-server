# SSH MCP Server Multi-Platform Build Script for PowerShell

param(
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 Building SSH MCP Server v$Version" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Yellow

# Clean previous builds
Write-Host "`n🧹 Cleaning previous builds..." -ForegroundColor Cyan
Remove-Item -Path "build" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "dist" -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path "build" -Force | Out-Null
New-Item -ItemType Directory -Path "dist" -Force | Out-Null

# Build configuration
$AppName = "sshmcp"
$Repo = "github.com/Cigarliu/ssh-mcp-server"
$BuildTime = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
$Ldflags = "-s -w -X ${Repo}/pkg/version.Version=${Version} -X ${Repo}/pkg/version.BuildTime=${BuildTime}"

# Supported platforms
$Platforms = @(
    @{ OS = "windows"; Arch = "amd64"; Ext = ".exe" },
    @{ OS = "windows"; Arch = "386"; Ext = ".exe" },
    @{ OS = "windows"; Arch = "arm64"; Ext = ".exe" },
    @{ OS = "linux"; Arch = "amd64"; Ext = "" },
    @{ OS = "linux"; Arch = "arm64"; Ext = "" },
    @{ OS = "linux"; Arch = "386"; Ext = "" },
    @{ OS = "linux"; Arch = "arm"; Ext = "" },
    @{ OS = "darwin"; Arch = "amd64"; Ext = "" },
    @{ OS = "darwin"; Arch = "arm64"; Ext = "" }
)

Write-Host "`n📦 Building binaries for $($Platforms.Count) platforms...`n" -ForegroundColor Cyan

foreach ($Platform in $Platforms) {
    $GOOS = $Platform.OS
    $GOARCH = $Platform.Arch
    $Ext = $Platform.Ext
    $OutputName = "${AppName}-${GOOS}-${GOARCH}${Ext}"

    Write-Host "Building ${OutputName}..." -ForegroundColor Yellow

    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    $env:CGO_ENABLED = "0"

    go build -ldflags $Ldflags -o "build/${OutputName}" ./cmd/server

    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Failed to build ${OutputName}" -ForegroundColor Red
        exit 1
    }

    # Create archive
    Set-Location build
    if ($GOOS -eq "windows") {
        $ArchiveName = "${OutputName}-${Version}.zip"
        Compress-Archive -Path $OutputName -DestinationPath "../dist/${ArchiveName}"
    } else {
        $ArchiveName = "${OutputName}-${Version}.tar.gz"
        if ($IsWindows) {
            # On Windows, use tar if available (Windows 10+)
            tar czf "../dist/${ArchiveName}" $OutputName
        } else {
            # On Unix-like systems
            & tar czf "../dist/${ArchiveName}" $OutputName
        }
    }
    Set-Location ..

    Write-Host "✅ ${OutputName} built successfully" -ForegroundColor Green
}

# Generate checksums
Write-Host "`n🔐 Generating checksums..." -ForegroundColor Cyan
$ChecksumFile = "dist/checksums.txt"
Get-FileHash dist/* -Algorithm SHA256 | ForEach-Object {
    $Hash = $_.Hash.ToLower()
    $File = [System.IO.Path]::GetFileName($_.Path)
    "${Hash}  ${File}" | Out-File -Append $ChecksumFile
}

Write-Host "`n✨ Build completed!" -ForegroundColor Green
Write-Host "📁 Binaries are in: dist/`n" -ForegroundColor Cyan
Write-Host "📋 Generated files:" -ForegroundColor Yellow
Get-ChildItem -Path dist | ForEach-Object {
    Write-Host "  $($_.Name) ($([math]::Round($_.Length/1KB, 2)) KB)" -ForegroundColor White
}

Write-Host "`n🔐 Checksums:" -ForegroundColor Yellow
Get-Content $ChecksumFile | ForEach-Object { Write-Host "  $_" -ForegroundColor White }

Write-Host "`n🎯 Ready to create GitHub Release!`n" -ForegroundColor Green
Write-Host "To create a release:" -ForegroundColor Cyan
Write-Host "  1. git tag v$Version" -ForegroundColor White
Write-Host "  2. git push origin v$Version" -ForegroundColor White
Write-Host "  3. gh release create v$Version --title 'v$Version' --notes 'See CHANGELOG.md'" -ForegroundColor White
Write-Host "  4. gh release upload v$Version dist/*`n" -ForegroundColor White
