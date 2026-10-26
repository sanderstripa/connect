$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Drawing
$connectRoot=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
if(-not ('ConnectBrandRenderer' -as [type])) { Add-Type -Path (Join-Path $PSScriptRoot 'BrandRenderer.cs') -ReferencedAssemblies (@('System.Drawing.Common','System.Drawing.Primitives','System.Xml.ReaderWriter','System.Xml.XmlDocument','System.Text.RegularExpressions','System.ComponentModel.Primitives') + @(Get-ChildItem $PSHOME -Filter 'System.Private.Windows.*.dll' | ForEach-Object FullName)) }
$connectSvg=Join-Path $connectRoot 'build/branding/connect.svg'
$appSource=Join-Path $connectRoot 'build/branding/source/app-round.png'
$trayOffSource=Join-Path $connectRoot 'build/branding/source/tray-off.png'
$trayOnSource=Join-Path $connectRoot 'build/branding/source/tray-on.png'
foreach($size in @(16,20,24,28,32,36,40,48,64,96,128,256,512,1024)){
 [ConnectBrandRenderer]::Resize($appSource,(Join-Path $connectRoot "build/branding/icon-$size.png"),$size)
}
foreach($size in @(16,20,24,28,32,36,40,48,64,96,128,256)){
 [ConnectBrandRenderer]::Resize($trayOffSource,(Join-Path $connectRoot "build/tray/off-$size.png"),$size)
 [ConnectBrandRenderer]::Resize($trayOnSource,(Join-Path $connectRoot "build/tray/on-$size.png"),$size)
}
foreach($pair in @(@('build/branding/icon-1024.png','build/appicon.png'),@('build/branding/icon-256.png','cmd/installer/web/assets/icon.png'),@('build/branding/icon-512.png','frontend/src/assets/images/logo-universal.png'),@('build/branding/icon-128.png','extension/assets/icon.png'),@('build/tray/off-32.png','build/tray/off-light.png'),@('build/tray/off-32.png','build/tray/off-dark.png'),@('build/tray/on-32.png','build/tray/on.png'))){Copy-Item (Join-Path $connectRoot $pair[0]) (Join-Path $connectRoot $pair[1]) -Force}
foreach($size in @(16,24,32,48,64,128)){Copy-Item (Join-Path $connectRoot "build/branding/icon-$size.png") (Join-Path $connectRoot "extension/assets/icon-$size.png") -Force}
Copy-Item (Join-Path $connectRoot 'build/branding/icon-256.png') (Join-Path $connectRoot 'installer-core/products/vpn.png') -Force
Copy-Item $connectSvg (Join-Path $connectRoot 'build/tray/logo.svg') -Force
$mono=[IO.File]::ReadAllText($connectSvg) -replace '(?s)<defs>.*?</defs>','' -replace 'fill="url\(#background\)"','fill="none"' -replace 'url\(#silver\)','currentColor'
[IO.File]::WriteAllText((Join-Path $connectRoot 'build/tray/logo-monochrome.svg'),[regex]::Replace($mono,'(?m)[ \t]+(?=\r?$)',''),[Text.UTF8Encoding]::new($false))
