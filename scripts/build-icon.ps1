$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot 'build-branding.ps1')
& node (Join-Path $PSScriptRoot 'package-icons.mjs')
Copy-Item (Join-Path $PSScriptRoot '../build/windows/icon.ico') (Join-Path $PSScriptRoot '../installer-core/products/vpn.ico') -Force
if($LASTEXITCODE -ne 0){throw 'Icon packaging failed'}
