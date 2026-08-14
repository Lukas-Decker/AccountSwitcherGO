# Stages a shippable build in dist/ and zips it.
#
# The application embeds its frontend, Platforms.json, GameStats.json, locales
# and tray icon, so a release is the executable plus the licence texts it has to
# carry when redistributed. Nothing else beside the exe is read at runtime, and
# per-user data lives in %AppData%, not next to the binary.
#
# Kept as a script rather than inline Taskfile commands: staging is a sequence of
# steps that has to fail loudly and clean up after itself, which is unreadable
# once escaped through YAML twice.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Root,
    [Parameter(Mandatory = $true)][string]$BinDir,
    [Parameter(Mandatory = $true)][string]$BuiltExe,
    [string]$Arch = "amd64"
)

$ErrorActionPreference = "Stop"

# The name the app ships under. The build task emits the hyphenated APP_NAME
# internally; users see the product name.
$ShippedExe = "AccountSwitcher.exe"

# Redistribution needs these; everything else in the repo is source or dev tooling.
$Payload = @("LICENSE", "OPEN_SOURCE_LICENSES.txt")

function Get-AppVersion {
    param([string]$ConfigPath)
    if (-not (Test-Path $ConfigPath)) { return "0.0.0" }
    # The version sits under `info:` in build/config.yml. Matched directly rather
    # than by parsing YAML, to avoid a module dependency for one value.
    foreach ($line in Get-Content $ConfigPath) {
        if ($line -match '^\s*version:\s*"([^"]+)"') { return $Matches[1] }
    }
    return "0.0.0"
}

$srcExe = Join-Path $Root (Join-Path $BinDir $BuiltExe)
if (-not (Test-Path $srcExe)) {
    throw "dist: no built executable at $srcExe. Run 'wails3 task build' first."
}

$version = Get-AppVersion (Join-Path $Root "build\config.yml")
$distRoot = Join-Path $Root "dist"
$stageName = "AccountSwitcher-$version-windows-$Arch"
$stageDir = Join-Path $distRoot $stageName
$zipPath = Join-Path $distRoot "$stageName.zip"

# Rebuilt from scratch every time, so a file dropped from the payload cannot
# linger in a release from an earlier run.
if (Test-Path $distRoot) { Remove-Item -Recurse -Force -LiteralPath $distRoot }
New-Item -ItemType Directory -Force -Path $stageDir | Out-Null

Copy-Item -LiteralPath $srcExe -Destination (Join-Path $stageDir $ShippedExe) -Force
foreach ($name in $Payload) {
    $src = Join-Path $Root $name
    if (Test-Path $src) {
        Copy-Item -LiteralPath $src -Destination (Join-Path $stageDir $name) -Force
    } else {
        Write-Warning "dist: $name not found, skipped"
    }
}

# Compresses the staged folder itself, so the archive expands into a directory
# rather than scattering an exe into whatever folder the user opened it from.
Compress-Archive -Path $stageDir -DestinationPath $zipPath -CompressionLevel Optimal -Force

$exeSize = [math]::Round((Get-Item (Join-Path $stageDir $ShippedExe)).Length / 1MB, 1)
$zipSize = [math]::Round((Get-Item $zipPath).Length / 1MB, 1)
Write-Host ""
Write-Host "dist: staged $stageName"
foreach ($f in Get-ChildItem $stageDir) {
    Write-Host ("       {0,-32} {1,8:N0} bytes" -f $f.Name, $f.Length)
}
Write-Host ("dist: exe {0} MB, zip {1} MB" -f $exeSize, $zipSize)
Write-Host "dist: $zipPath"
