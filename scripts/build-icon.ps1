$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$source = [Drawing.Image]::FromFile((Join-Path $root 'build/appicon.png'))
$images = @()
try {
 foreach ($size in @(16,24,32,48,64,128,256)) {
  $bmp = New-Object Drawing.Bitmap $size,$size
  $g = [Drawing.Graphics]::FromImage($bmp)
  $g.InterpolationMode = [Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $g.DrawImage($source,0,0,$size,$size)
  $stream = New-Object IO.MemoryStream
  $bmp.Save($stream,[Drawing.Imaging.ImageFormat]::Png)
  $images += [pscustomobject]@{Size=$size;Bytes=$stream.ToArray()}
  $stream.Dispose(); $g.Dispose(); $bmp.Dispose()
 }
 $stream = [IO.File]::Create((Join-Path $root 'build/windows/icon.ico'))
 $writer = New-Object IO.BinaryWriter $stream
 try {
  $writer.Write([uint16]0);$writer.Write([uint16]1);$writer.Write([uint16]$images.Count)
  $offset=6+16*$images.Count
  foreach($im in $images) {
   $dim=if($im.Size -eq 256){0}else{$im.Size}
   $writer.Write([byte]$dim);$writer.Write([byte]$dim);$writer.Write([byte]0);$writer.Write([byte]0)
   $writer.Write([uint16]1);$writer.Write([uint16]32);$writer.Write([uint32]$im.Bytes.Length);$writer.Write([uint32]$offset)
   $offset += $im.Bytes.Length
  }
  foreach($im in $images){$writer.Write([byte[]]$im.Bytes)}
 } finally {$writer.Dispose();$stream.Dispose()}
} finally {$source.Dispose()}
