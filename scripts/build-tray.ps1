$ErrorActionPreference='Stop'
# Tray states use the approved transparent PNG masters in build/branding/source/.
if(-not (Test-Path (Join-Path $PSScriptRoot '../build/tray/off-16.png'))){& (Join-Path $PSScriptRoot 'build-branding.ps1')}
