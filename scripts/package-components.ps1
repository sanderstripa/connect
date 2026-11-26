$ErrorActionPreference='Stop'
$connectRoot=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$connectBin=Join-Path $connectRoot 'runtime/bin/windows-amd64'
$connectManifest=[ordered]@{}
$connectXray=& (Join-Path $connectBin 'xray.exe') version | Select-Object -First 1
if($LASTEXITCODE -ne 0 -or $connectXray -notmatch '^Xray\s+(\S+)'){throw 'Cannot identify Xray version'}
$connectManifest['xray.exe']=@{version=('v'+$Matches[1]);sha256=(Get-FileHash (Join-Path $connectBin 'xray.exe') -Algorithm SHA256).Hash.ToLowerInvariant()}
$connectSudoku=& go version -m (Join-Path $connectBin 'sudoku.exe')
$connectSudokuVersion=($connectSudoku | Select-String '^\s+mod\s+github.com/SUDOKU-ASCII/sudoku\s+(\S+)' | Select-Object -First 1).Matches.Groups[1].Value
if(-not $connectSudokuVersion){throw 'Cannot identify Sudoku version'}
# Go records source adapters as +dirty; package names use the protocol release.
$connectSudokuVersion=($connectSudokuVersion -split '\+')[0]
if($connectSudokuVersion -ne 'v0.5.2-1'){throw 'Unexpected bundled Sudoku protocol version'}
$connectAdapterVersion=(Get-Content (Join-Path $connectRoot 'frontend/package.json') -Raw | ConvertFrom-Json).version
$connectManifest['sudoku.exe']=@{version=($connectSudokuVersion+'+connect.'+$connectAdapterVersion);sha256=(Get-FileHash (Join-Path $connectBin 'sudoku.exe') -Algorithm SHA256).Hash.ToLowerInvariant()}
$connectAWG=& (Join-Path $connectBin 'awg-client.exe') -version
if($LASTEXITCODE -ne 0 -or -not $connectAWG){throw 'Cannot identify AmneziaWG version'}
$connectManifest['awg-client.exe']=@{version=('v'+$connectAdapterVersion);build=$connectAWG;sha256=(Get-FileHash (Join-Path $connectBin 'awg-client.exe') -Algorithm SHA256).Hash.ToLowerInvariant()}
$geoVersionFile=Join-Path $connectBin 'geo-version.txt'
$geoVersion=(Get-Content -LiteralPath $geoVersionFile -Raw).Trim()
if($geoVersion -notmatch '^\d{12}$'){throw 'Cannot identify GeoIP / GeoSite version'}
foreach($name in @('geoip.dat','geosite.dat')) {
 $digest=(Get-FileHash -LiteralPath (Join-Path $connectBin $name) -Algorithm SHA256).Hash.ToLowerInvariant()
 $connectManifest[$name]=@{version=$geoVersion;sha256=$digest}
}
$hevVersion=(Get-Content -LiteralPath (Join-Path $connectBin 'hev-version.txt') -Raw).Trim()
if($hevVersion -notmatch '^\d+\.\d+\.\d+$'){throw 'Cannot identify HEV version'}
foreach($name in @('hev-socks5-tunnel.exe','wintun.dll')) {
 $digest=(Get-FileHash -LiteralPath (Join-Path $connectBin $name) -Algorithm SHA256).Hash.ToLowerInvariant()
 $connectManifest[$name]=@{version=$hevVersion;sha256=$digest}
}
$connectManifest | ConvertTo-Json -Depth 4 | Set-Content (Join-Path $connectBin 'components.json') -Encoding utf8
New-Item -ItemType Directory -Force (Join-Path $connectRoot 'build/bin') | Out-Null

Compress-Archive -LiteralPath (Join-Path $connectBin 'awg-client.exe') -DestinationPath (Join-Path $connectRoot 'build/bin/Connect-AmneziaWG-windows-amd64.zip') -Force

Compress-Archive -LiteralPath (Join-Path $connectBin 'sudoku.exe') -DestinationPath (Join-Path $connectRoot ("build/bin/Connect-Sudoku-"+$connectSudokuVersion+"-windows-amd64.zip")) -Force
