$ErrorActionPreference='Stop'
# Preserve the per-size vector exports and optical stroke adjustments. The old
# bitmap resizer blurred small glyphs. Regenerate explicitly with
# node scripts/render-app-icons.cjs (development dependency: sharp).
& node (Join-Path $PSScriptRoot 'verify-branding.mjs')
if($LASTEXITCODE -ne 0){throw 'Approved icon exports are incomplete or inconsistent'}
