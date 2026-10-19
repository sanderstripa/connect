param([string]$OutputDirectory=(Join-Path $PSScriptRoot '../.tmp/installer-review'))
$ErrorActionPreference='Stop'
Push-Location $PSScriptRoot
try{
 if(!$env:PLAYWRIGHT_MODULE){
  & npm.cmd install --ignore-scripts --no-audit --no-fund
  if($LASTEXITCODE){throw 'Playwright dependency installation failed'}
 }
 $env:INSTALLER_REVIEW_DIR=[IO.Path]::GetFullPath($OutputDirectory)
 & node ui-review.cjs
 if($LASTEXITCODE){throw 'Shared installer UI checks failed'}
}finally{Pop-Location}
