param(
 [Parameter(Mandatory=$true)][ValidateSet('vpn','screenshot','screencast','torrent','soulu')][string]$Product,
 [Parameter(Mandatory=$true)][string]$PayloadPath,
 [Parameter(Mandatory=$true)][string]$Version,
 [Parameter(Mandatory=$true)][string]$OutFile,
 [string]$ExtensionPath,[string]$ExpectedExecutable,[string]$ProductCommit='',
 [string]$Go='go',[switch]$SkipTests
)
$ErrorActionPreference='Stop'
if($Version -notmatch '^\d+\.\d+\.\d+([+-][A-Za-z0-9.-]+)?$'){throw 'Invalid product version'}
$payload=(Resolve-Path -LiteralPath $PayloadPath).Path
$output=[IO.Path]::GetFullPath($OutFile)
$adapter=Join-Path $PSScriptRoot "adapters/$Product"
$env:CGO_ENABLED='0'
function Native([string]$Tool,[string[]]$Arguments){& $Tool @Arguments;if($LASTEXITCODE){throw "Build step failed: $Tool"}}
switch($Product){
 'vpn'{
  if(!$ExtensionPath){throw 'VPN requires its matching extension source directory'}
  New-Item -ItemType Directory -Force "$adapter/payload","$adapter/chrome" | Out-Null
  Copy-Item -LiteralPath $payload -Destination "$adapter/payload/Connect.exe" -Force
  Copy-Item -Path (Join-Path (Resolve-Path $ExtensionPath) '*') -Destination "$adapter/chrome" -Recurse -Force
 }
 'screenshot'{Copy-Item -LiteralPath $payload -Destination "$adapter/Screenshot.exe" -Force}
 'torrent'{
  if(!$ExpectedExecutable){throw 'Torrent requires the matching application executable for payload verification'}
  $expectedHash=(Get-FileHash -LiteralPath $ExpectedExecutable -Algorithm SHA256).Hash.ToLowerInvariant()
  Copy-Item -LiteralPath $payload -Destination "$adapter/payload.exe" -Force
 }
 default{
  if((Get-Item -LiteralPath $payload).PSIsContainer){Compress-Archive -Path (Join-Path $payload '*') -DestinationPath "$adapter/payload.zip" -Force}
  else{Copy-Item -LiteralPath $payload -Destination "$adapter/payload.zip" -Force}
 }
}
New-Item -ItemType Directory -Force ([IO.Path]::GetDirectoryName($output)) | Out-Null
Push-Location $PSScriptRoot
try{
 $metadata="-X connect-installer.local/core/products.VersionOverride=$Version"
 if($Product -eq 'torrent'){$metadata+=" -X connect-installer.local/core/adapters/torrent.payloadSHA=$expectedHash"}
 $previousCiTest=$env:CONNECT_INSTALLER_CI_TEST
 try{
  if($env:GITHUB_ACTIONS -eq 'true'){$env:CONNECT_INSTALLER_CI_TEST='1'}
  if(!$SkipTests){Native $Go @('test','-v',"-ldflags=$metadata",'./core','./installprefs','./products',"./adapters/$Product")}
 }finally{$env:CONNECT_INSTALLER_CI_TEST=$previousCiTest}
 Native $Go @('run','./tools/resources',"products/$Product.ico","products/$Product.manifest","cmd/$Product/resources_windows_amd64.syso")
 $flags="-w -s -H windowsgui $metadata"
 Native $Go @('build','-tags','production','-trimpath','-buildvcs=false',"-ldflags=$flags",'-o',$output,"./cmd/$Product")
 $coreCommit=(& git -C $PSScriptRoot rev-parse HEAD).Trim()
 $payloadFile=if((Get-Item $payload).PSIsContainer){"$adapter/payload.zip"}else{$payload}
 [ordered]@{product=$Product;version=$Version;productCommit=$ProductCommit;coreCommit=$coreCommit;sha256=(Get-FileHash -LiteralPath $output).Hash.ToLowerInvariant();payloadSHA256=(Get-FileHash $payloadFile).Hash.ToLowerInvariant()} | ConvertTo-Json | Set-Content -LiteralPath "$output.build.json" -Encoding utf8
}finally{Pop-Location}
