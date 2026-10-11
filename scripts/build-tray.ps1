Add-Type -AssemblyName System.Drawing
$connectTrayDir=Join-Path $PSScriptRoot '../build/tray'
# Draw the application's concentric rings and wave mark at high resolution,
# then downsample once for crisp tray edges. Background stays black in both states.
foreach($connectVariant in @(@{Name='off-light';Color='#B4B7BB'},@{Name='off-dark';Color='#B4B7BB'},@{Name='on';Color='#26BC79'})) {
 $connectCanvas=[Drawing.Bitmap]::new(256,256)
 $connectGraphics=[Drawing.Graphics]::FromImage($connectCanvas)
 $connectGraphics.SmoothingMode=[Drawing.Drawing2D.SmoothingMode]::AntiAlias
 $connectBrush=[Drawing.SolidBrush]::new([Drawing.Color]::FromArgb(255,14,15,16))
 $connectGraphics.FillEllipse($connectBrush,5,5,246,246)
 $connectPen=[Drawing.Pen]::new([Drawing.ColorTranslator]::FromHtml($connectVariant.Color),9)
 $connectPen.StartCap=[Drawing.Drawing2D.LineCap]::Round
 $connectPen.EndCap=[Drawing.Drawing2D.LineCap]::Round
 $connectGraphics.DrawEllipse($connectPen,10,10,236,236)
 $connectGraphics.DrawEllipse($connectPen,49,49,158,158)
 $connectGraphics.DrawArc($connectPen,85,85,86,86,155,245)
 $connectPath=[Drawing.Drawing2D.GraphicsPath]::new()
 $connectPath.AddBezier(51,158,87,185,106,90,146,137)
 $connectPath.AddBezier(146,137,162,154,173,173,188,187)
 $connectGraphics.DrawPath($connectPen,$connectPath)
 $connectGraphics.DrawBezier($connectPen,90,173,119,147,126,152,151,181)
 $connectSmall=[Drawing.Bitmap]::new(32,32)
 $connectScale=[Drawing.Graphics]::FromImage($connectSmall)
 $connectScale.InterpolationMode=[Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
 $connectScale.PixelOffsetMode=[Drawing.Drawing2D.PixelOffsetMode]::HighQuality
 $connectScale.DrawImage($connectCanvas,0,0,32,32)
 $connectSmall.Save((Join-Path $connectTrayDir ($connectVariant.Name+'.png')),[Drawing.Imaging.ImageFormat]::Png)
 $connectScale.Dispose();$connectSmall.Dispose();$connectPath.Dispose();$connectGraphics.Dispose();$connectPen.Dispose();$connectBrush.Dispose();$connectCanvas.Dispose()
}
