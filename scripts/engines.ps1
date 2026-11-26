$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$out=Join-Path $root 'runtime/bin/windows-amd64'
New-Item -ItemType Directory -Force $out | Out-Null
$scratch=Join-Path $root '.tmp/xray-v26.11.9'
New-Item -ItemType Directory -Force $scratch | Out-Null
$archive=Join-Path $scratch 'Xray-windows-64.zip'
Invoke-WebRequest 'https://github.com/XTLS/Xray-core/releases/download/v26.11.9/Xray-windows-64.zip' -OutFile $archive
$expected='1262522266337b824f07beb8e1375786b437683b3890105469088f8e17e0a69d'
if((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash -ne $expected){throw 'Xray archive checksum mismatch'}
Expand-Archive -LiteralPath $archive -DestinationPath $scratch -Force
Copy-Item -LiteralPath (Join-Path $scratch 'xray.exe') -Destination $out
Copy-Item -LiteralPath (Join-Path $scratch 'LICENSE') -Destination (Join-Path $out 'Xray-LICENSE.txt')
$geoVersion='202611252204'
$geoFiles=@{
 'geoip.dat'='d4ec4c5e5d5a38c1399964ea9fd8c94288426a3b31b483904b67819fc07f4466'
 'geosite.dat'='ba10794ee0d5584176b574694ceb508316e9080cd8b9c3e6fd62f441e044f16e'
}
foreach($name in $geoFiles.Keys) {
 $file=Join-Path $scratch $name
 Invoke-WebRequest "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/download/$geoVersion/$name" -OutFile $file
 if((Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash -ne $geoFiles[$name]){throw "$name checksum mismatch"}
}
foreach($name in $geoFiles.Keys) {
 Copy-Item -LiteralPath (Join-Path $scratch $name) -Destination $out
}
Set-Content -LiteralPath (Join-Path $out 'geo-version.txt') -Value $geoVersion -Encoding ascii
Push-Location $root
try {
 & go test ./internal/awgconfig ./cmd/awg-client
 if($LASTEXITCODE -ne 0){throw 'AmneziaWG tests failed'}
 $env:CGO_ENABLED='0'
 & go build -trimpath '-ldflags=-w -s' -o (Join-Path $out 'awg-client.exe') ./cmd/awg-client
 if($LASTEXITCODE -ne 0){throw 'AmneziaWG build failed'}
 Get-ChildItem -LiteralPath (Join-Path $root 'build/licenses') -Filter 'AmneziaWG-*.txt' | Copy-Item -Destination $out
}finally{Pop-Location}
