$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Drawing
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
foreach($size in @(16,20,24,28,32,36,40,48,64,96,128,256)) {
 foreach($kind in @('app','off','on')) {
  $file=if($kind -eq 'app'){"build/branding/icon-$size.png"}else{"build/tray/$kind-$size.png"}
  $image=[Drawing.Bitmap]::new((Join-Path $root $file))
  try {
   if($image.Width -ne $size -or $image.Height -ne $size){throw "Invalid size: $file"}
   $transparent=0;$opaque=0
   for($y=0;$y -lt $size;$y++){for($x=0;$x -lt $size;$x++){
    $p=$image.GetPixel($x,$y)
    if($p.A -eq 0){$transparent++}
    if($p.A -gt 240){$opaque++}
   }}
   foreach($xy in @(@(0,0),@(($size-1),0),@(0,($size-1)),@(($size-1),($size-1)))){
    if($image.GetPixel($xy[0],$xy[1]).A -ne 0){throw "Opaque background corner: $file"}
   }
   if($opaque -eq 0 -or $transparent -lt ($size*$size*0.15)){throw "Missing artwork or transparency: $file"}
   if($kind -ne 'app' -and $transparent -lt ($size*$size*0.35)){throw "Tray may contain a plate: $file"}
  } finally {$image.Dispose()}
 }
}
Write-Host 'PASS: all app and tray sizes have visible artwork and transparent backgrounds'
