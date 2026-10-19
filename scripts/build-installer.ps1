param([switch]$SkipApp)
$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Push-Location $root
try{
 if(!$SkipApp){& ./scripts/build-windows.ps1}
 $version=(Get-Content frontend/package.json -Raw | ConvertFrom-Json).version
 Compress-Archive -Path extension/* -DestinationPath build/bin/Connect-Chrome.zip -Force
 & ./scripts/build-connect-installer.ps1 -PayloadPath (Join-Path $root 'build/bin/Connect.exe') -Version $version -OutFile (Join-Path $root 'build/bin/Connect-Setup.exe') -ExtensionPath (Join-Path $root 'extension')
}finally{Pop-Location}
