Add-Type -AssemblyName System.Drawing
$connectTrayDir=Join-Path $PSScriptRoot '../build/tray'
foreach($connectVariant in @(@{Name='off-light';Color='#505762'},@{Name='off-dark';Color='#E6E9EE'},@{Name='on';Color='#26BC79'})) {
 $connectCanvas=[Drawing.Bitmap]::new(256,256)
 $connectGraphics=[Drawing.Graphics]::FromImage($connectCanvas)
 $connectGraphics.SmoothingMode=[Drawing.Drawing2D.SmoothingMode]::AntiAlias
 $connectPen=[Drawing.Pen]::new([Drawing.ColorTranslator]::FromHtml($connectVariant.Color),22)
 $connectPen.StartCap=[Drawing.Drawing2D.LineCap]::Round
 $connectPen.EndCap=[Drawing.Drawing2D.LineCap]::Round
 $connectGraphics.DrawArc($connectPen,45,51,166,166,-52,284)
 $connectGraphics.DrawLine($connectPen,128,27,128,122)
 $connectSmall=[Drawing.Bitmap]::new(32,32)
 $connectScale=[Drawing.Graphics]::FromImage($connectSmall)
 $connectScale.InterpolationMode=[Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
 $connectScale.PixelOffsetMode=[Drawing.Drawing2D.PixelOffsetMode]::HighQuality
 $connectScale.DrawImage($connectCanvas,0,0,32,32)
 $connectSmall.Save((Join-Path $connectTrayDir ($connectVariant.Name+'.png')),[Drawing.Imaging.ImageFormat]::Png)
 $connectScale.Dispose();$connectSmall.Dispose();$connectGraphics.Dispose();$connectPen.Dispose();$connectCanvas.Dispose()
}
