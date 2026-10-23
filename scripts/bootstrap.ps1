# Recreates the h3xdOS build environment on a fresh Windows machine: checks the
# WSL platform, installs the h3xdos-build distro on A:, then provisions it.
# It never elevates itself; steps that need an administrator are printed instead.
$ErrorActionPreference = 'Stop'

$ProjectRoot = 'A:\h3xdOS'
$DistroName = 'h3xdos-build'
$DistroDir = Join-Path $ProjectRoot 'wsl\h3xdos-build'
# Provisioning lives in a bash script because Windows PowerShell 5.1 mangles
# quotes in arguments to native programs and pipes CRLF into their stdin.
$ProvisionScript = '/mnt/a/h3xdOS/src/scripts/provision.sh'
$env:WSL_UTF8 = '1'

if (-not (Test-Path -LiteralPath $ProjectRoot)) {
    Write-Host "$ProjectRoot not found. Clone the sources to $ProjectRoot\src first."
    exit 1
}

$wsl = Get-Command wsl.exe -ErrorAction SilentlyContinue
if ($wsl) {
    $null = & wsl.exe --version
}
if (-not $wsl -or $LASTEXITCODE -ne 0) {
    Write-Host 'WSL 2 is not installed. In an elevated PowerShell run:'
    Write-Host '  wsl --install --no-distribution'
    Write-Host '  wsl --set-default-version 2'
    Write-Host 'Reboot, then run bootstrap.ps1 again from a normal PowerShell.'
    exit 1
}

$distros = & wsl.exe --list --quiet
# wsl --list exits non-zero when no distro is registered at all.
if ($LASTEXITCODE -ne 0) {
    $distros = @()
}
$distros = @($distros | ForEach-Object { $_.Trim() })

if ($distros -notcontains $DistroName) {
    $vhdx = Join-Path $DistroDir 'ext4.vhdx'
    if (Test-Path -LiteralPath $vhdx) {
        Write-Host "$vhdx exists but $DistroName is not registered. Register the existing disk with:"
        Write-Host "  wsl --import-in-place $DistroName $vhdx"
        exit 1
    }
    New-Item -ItemType Directory -Force -Path (Split-Path $DistroDir) | Out-Null
    & wsl.exe --install archlinux --name $DistroName --location $DistroDir --no-launch
    if ($LASTEXITCODE -ne 0) {
        throw "wsl --install failed with exit code $LASTEXITCODE"
    }
}

& wsl.exe -d $DistroName -u root --exec bash $ProvisionScript
if ($LASTEXITCODE -ne 0) {
    throw "provision.sh failed with exit code $LASTEXITCODE"
}

Write-Host "$DistroName is ready. Build the ISO with:"
Write-Host "  wsl.exe -d $DistroName -u root -- bash /mnt/a/h3xdOS/src/scripts/build.sh"
