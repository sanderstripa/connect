param([switch]$SkipApp)
$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Push-Location $root
try {
 if(-not $SkipApp){& ./scripts/build-windows.ps1}
 New-Item -ItemType Directory -Force cmd/installer/payload,cmd/installer/chrome | Out-Null
 Copy-Item -Path extension/* -Destination cmd/installer/chrome -Recurse -Force
 Compress-Archive -Path extension/* -DestinationPath build/bin/Connect-Chrome.zip -Force
 Copy-Item -LiteralPath build/bin/Connect.exe -Destination cmd/installer/payload/Connect.exe
 if(Test-Path -LiteralPath cmd/installer/wails_windows_amd64.syso){Remove-Item -LiteralPath cmd/installer/wails_windows_amd64.syso}
 & go test ./cmd/installer
 if($LASTEXITCODE -ne 0){throw 'Installer tests failed'}
 & go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.74 generate syso -arch amd64 -icon build/windows/icon.ico -manifest build/windows/installer.exe.manifest -info build/windows/info.json -out cmd/installer/wails_windows_amd64.syso
 if($LASTEXITCODE -ne 0){throw 'Installer resources failed'}
 $env:CGO_ENABLED='0'
 & go build -tags production -trimpath -buildvcs=false '-ldflags=-w -s -H windowsgui' -o build/bin/Connect-Setup.exe ./cmd/installer
 if($LASTEXITCODE -ne 0){throw 'Installer build failed'}
}finally{Pop-Location}
