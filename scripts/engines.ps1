$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$out=Join-Path $root 'runtime/bin/windows-amd64'
New-Item -ItemType Directory -Force $out | Out-Null
$scratch=Join-Path $root '.tmp/xray-v26.3.27'
New-Item -ItemType Directory -Force $scratch | Out-Null
$archive=Join-Path $scratch 'Xray-windows-64.zip'
Invoke-WebRequest 'https://github.com/XTLS/Xray-core/releases/download/v26.3.27/Xray-windows-64.zip' -OutFile $archive
$expected='D004C39288CE9ADA487C6F398C7C545F7D749E44BDFDD59DBC9F865AFBA4E1AD'
if((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash -ne $expected){throw 'Xray archive checksum mismatch'}
Expand-Archive -LiteralPath $archive -DestinationPath $scratch -Force
Copy-Item -LiteralPath (Join-Path $scratch 'xray.exe') -Destination $out
Copy-Item -LiteralPath (Join-Path $scratch 'LICENSE') -Destination (Join-Path $out 'Xray-LICENSE.txt')
Push-Location (Join-Path $root 'engines/mist')
try {
 & go test ./cmd/mist-client ./internal/config
 if($LASTEXITCODE -ne 0){throw 'Mist tests failed'}
 $env:CGO_ENABLED='0'
 & go build -trimpath '-ldflags=-w -s' -o (Join-Path $out 'mist-client.exe') ./cmd/mist-client
 if($LASTEXITCODE -ne 0){throw 'Mist build failed'}
}finally{Pop-Location}
