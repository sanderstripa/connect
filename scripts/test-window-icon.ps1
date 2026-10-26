# Disposable Windows test: check the actual application's window icons.
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Drawing
$drawingReferences=if($PSVersionTable.PSVersion.Major -ge 6){
 @('System.Drawing.Common','System.Drawing.Primitives','System.ComponentModel.Primitives') + @(Get-ChildItem $PSHOME -Filter 'System.Private.Windows.*.dll' | ForEach-Object FullName)
}else{@('System.Drawing')}
Add-Type -ReferencedAssemblies $drawingReferences @'
using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.Runtime.InteropServices;
using System.Text;
public static class ConnectWindowIconProbe {
 delegate bool Callback(IntPtr h, IntPtr p);
 [DllImport("user32.dll")] static extern bool EnumWindows(Callback cb, IntPtr p);
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
 [DllImport("user32.dll")] static extern IntPtr SendMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
 [DllImport("user32.dll")] static extern bool DrawIconEx(IntPtr dc, int x, int y, IntPtr icon, int cx, int cy, uint step, IntPtr brush, uint flags);
 [DllImport("user32.dll")] static extern int GetSystemMetrics(int index);
 [DllImport("user32.dll")] static extern bool DestroyIcon(IntPtr icon);
 [DllImport("shell32.dll", CharSet=CharSet.Unicode)] static extern uint ExtractIconEx(string path, int index, out IntPtr large, out IntPtr small, uint count);
 public static IntPtr Find(uint processId) {
  IntPtr result=IntPtr.Zero;
  EnumWindows((h,p)=>{uint owner;GetWindowThreadProcessId(h,out owner);if(owner==processId){var title=new StringBuilder(256);GetWindowText(h,title,256);if(title.ToString()=="Connect"){result=h;return false;}}return true;},IntPtr.Zero);
  return result;
 }
 static Bitmap Render(IntPtr icon, int w, int h) {
  var bitmap=new Bitmap(w,h,PixelFormat.Format32bppArgb);
  using(var graphics=Graphics.FromImage(bitmap)){
   graphics.Clear(Color.White);
   var dc=graphics.GetHdc();
   try {if(!DrawIconEx(dc,0,0,icon,w,h,0,IntPtr.Zero,3))throw new InvalidOperationException("DrawIconEx failed");}
   finally {graphics.ReleaseHdc(dc);}
  }
  return bitmap;
 }
 static bool Equal(IntPtr actual, IntPtr expected, int w, int h) {
  if(actual==IntPtr.Zero || expected==IntPtr.Zero)return false;
  using(var a=Render(actual,w,h))using(var b=Render(expected,w,h)){
   for(int y=0;y<h;y++)for(int x=0;x<w;x++)if(a.GetPixel(x,y).ToArgb()!=b.GetPixel(x,y).ToArgb())return false;
  }
  return true;
 }
 public static bool Matches(IntPtr window, string executable) {
  IntPtr large,small;
  if(ExtractIconEx(executable,0,out large,out small,1)==0)throw new InvalidOperationException("Executable icon missing");
  try {
   return Equal(SendMessage(window,0x007f,IntPtr.Zero,IntPtr.Zero),small,GetSystemMetrics(49),GetSystemMetrics(50)) &&
          Equal(SendMessage(window,0x007f,(IntPtr)1,IntPtr.Zero),large,GetSystemMetrics(11),GetSystemMetrics(12));
  } finally {if(large!=IntPtr.Zero)DestroyIcon(large);if(small!=IntPtr.Zero)DestroyIcon(small);}
 }
}
'@
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$executable=(Resolve-Path (Join-Path $root 'build/bin/Connect.exe')).Path
$sandbox=Join-Path $root '.tmp/window-icon-test'
New-Item -ItemType Directory -Force (Join-Path $sandbox 'roaming'),(Join-Path $sandbox 'local') | Out-Null
$oldAppData=$env:APPDATA; $oldLocalAppData=$env:LOCALAPPDATA
try {
 $env:APPDATA=Join-Path $sandbox 'roaming'; $env:LOCALAPPDATA=Join-Path $sandbox 'local'
 $process=Start-Process -FilePath $executable -PassThru -WindowStyle Hidden
} finally {$env:APPDATA=$oldAppData; $env:LOCALAPPDATA=$oldLocalAppData}
try {
 $deadline=(Get-Date).AddSeconds(90); $matched=$false
 do {
  $process.Refresh()
  if($process.HasExited){throw "Connect exited before its window icon was verified: $($process.ExitCode)"}
  $window=[ConnectWindowIconProbe]::Find([uint32]$process.Id)
  if($window -ne [IntPtr]::Zero){$matched=[ConnectWindowIconProbe]::Matches($window,$executable)}
  if(!$matched){Start-Sleep -Milliseconds 300}
 } while(!$matched -and (Get-Date) -lt $deadline)
 if(!$matched){throw 'Actual small or large window icon differs from the application EXE'}
 Write-Host 'PASS: actual small and large CONNECT window icons match the executable'
} finally {if(-not $process.HasExited){$process.Kill();$process.WaitForExit()}}
