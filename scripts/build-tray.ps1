$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Drawing
$connectRoot=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$connectTrayDir=Join-Path $connectRoot 'build/tray'
$connectSource=[Drawing.Bitmap]::new((Join-Path $connectRoot 'build/appicon.png'))
$connectOn=$connectSource.Clone()
try {
 # Preserve every contour, alpha value and black background from the app icon.
 # Tint only its silver highlights; no substitute drawing or new geometry.
 for($y=0;$y -lt $connectOn.Height;$y++){for($x=0;$x -lt $connectOn.Width;$x++){
  $c=$connectSource.GetPixel($x,$y);$v=[Math]::Max($c.R,[Math]::Max($c.G,$c.B))/255.0
  $amount=[Math]::Clamp(($v-0.16)/0.25,0.0,1.0)
  $r=[int][Math]::Round($c.R*(1-$amount)+$v*38*$amount)
  $g=[int][Math]::Round($c.G*(1-$amount)+$v*210*$amount)
  $b=[int][Math]::Round($c.B*(1-$amount)+$v*133*$amount)
  $connectOn.SetPixel($x,$y,[Drawing.Color]::FromArgb($c.A,$r,$g,$b))
 }}
 foreach($size in @(16,20,24,28,32,36,40,48,64)) {foreach($variant in @('off','on')) {
  $bmp=[Drawing.Bitmap]::new($size,$size,[Drawing.Imaging.PixelFormat]::Format32bppArgb)
  $g=[Drawing.Graphics]::FromImage($bmp)
  $g.CompositingMode=[Drawing.Drawing2D.CompositingMode]::SourceCopy
  $g.InterpolationMode=[Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $g.PixelOffsetMode=[Drawing.Drawing2D.PixelOffsetMode]::HighQuality
  $attributes=[Drawing.Imaging.ImageAttributes]::new()
  $attributes.SetWrapMode([Drawing.Drawing2D.WrapMode]::TileFlipXY)
  $image=if($variant -eq 'on'){$connectOn}else{$connectSource}
  $g.DrawImage($image,[Drawing.Rectangle]::new(0,0,$size,$size),0,0,$image.Width,$image.Height,[Drawing.GraphicsUnit]::Pixel,$attributes)
  $bmp.Save((Join-Path $connectTrayDir "$variant-$size.png"),[Drawing.Imaging.ImageFormat]::Png)
  $attributes.Dispose();$g.Dispose();$bmp.Dispose()
 }}
 Copy-Item (Join-Path $connectTrayDir 'off-32.png') (Join-Path $connectTrayDir 'off-light.png') -Force
 Copy-Item (Join-Path $connectTrayDir 'off-32.png') (Join-Path $connectTrayDir 'off-dark.png') -Force
 Copy-Item (Join-Path $connectTrayDir 'on-32.png') (Join-Path $connectTrayDir 'on.png') -Force
}finally{$connectOn.Dispose();$connectSource.Dispose()}
