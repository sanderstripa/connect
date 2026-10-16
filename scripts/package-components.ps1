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
$connectManifest['sudoku.exe']=@{version=$connectSudokuVersion;sha256=(Get-FileHash (Join-Path $connectBin 'sudoku.exe') -Algorithm SHA256).Hash.ToLowerInvariant()}
$connectMist=& (Join-Path $connectBin 'mist-client.exe') -version
if($LASTEXITCODE -ne 0 -or -not $connectMist){throw 'Cannot identify Mist version'}
$connectManifest['mist-client.exe']=@{version='v0.1.6';build=$connectMist;sha256=(Get-FileHash (Join-Path $connectBin 'mist-client.exe') -Algorithm SHA256).Hash.ToLowerInvariant()}
$connectManifest | ConvertTo-Json -Depth 4 | Set-Content (Join-Path $connectBin 'components.json') -Encoding utf8
New-Item -ItemType Directory -Force (Join-Path $connectRoot 'build/bin') | Out-Null
Compress-Archive -LiteralPath (Join-Path $connectBin 'mist-client.exe') -DestinationPath (Join-Path $connectRoot 'build/bin/Connect-Mist-windows-amd64.zip') -Force
