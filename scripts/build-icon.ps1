$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot 'build-branding.ps1')
& node (Join-Path $PSScriptRoot 'package-icons.mjs')
if($LASTEXITCODE -ne 0){throw 'Icon packaging failed'}
