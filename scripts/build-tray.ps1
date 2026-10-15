$ErrorActionPreference='Stop'
# All tray states and DPI sizes are generated from the same SVG as the app icon.
if(-not (Test-Path (Join-Path $PSScriptRoot '../build/tray/off-16.png'))){& (Join-Path $PSScriptRoot 'build-branding.ps1')}
