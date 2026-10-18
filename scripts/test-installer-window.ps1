# Disposable Windows CI only. Never performs an installation.
$ErrorActionPreference='Stop'
Add-Type @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class InstallerWindowProbe {
 delegate bool Callback(IntPtr h, IntPtr p);
 [DllImport("user32.dll")] static extern bool EnumWindows(Callback cb, IntPtr p);
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
 [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 public static IntPtr Find(uint processId) {
  IntPtr result=IntPtr.Zero;
  EnumWindows((h,p)=>{uint owner;GetWindowThreadProcessId(h,out owner);if(owner==processId){var title=new StringBuilder(256);GetWindowText(h,title,256);if(title.ToString()=="CONNECT VPN"){result=h;return false;}}return true;},IntPtr.Zero);
  return result;
 }
}
'@
New-Item -ItemType Directory -Force .tmp | Out-Null
$p=Start-Process -FilePath (Resolve-Path build/bin/Connect-Setup.exe).Path -PassThru -WindowStyle Hidden -RedirectStandardOutput .tmp/installer-stdout.log -RedirectStandardError .tmp/installer-stderr.log
try {
 $deadline=(Get-Date).AddSeconds(35)
 do {
  Start-Sleep -Milliseconds 300
  $p.Refresh()
  if($p.HasExited){throw "Installer exited before window creation: $($p.ExitCode)"}
  $window=[InstallerWindowProbe]::Find([uint32]$p.Id)
 } while($window -eq [IntPtr]::Zero -and (Get-Date) -lt $deadline)
 if($window -eq [IntPtr]::Zero){throw 'Native installer window was not created'}
 Write-Host 'Native CONNECT VPN installer window created'
 # A native handle precedes asynchronous WebView2 initialization.
 Start-Sleep -Seconds 4
 $p.Refresh()
 if($p.HasExited){throw "Installer exited during WebView2 initialization: $($p.ExitCode)"}
 if(-not [InstallerWindowProbe]::PostMessage($window,0x0010,[IntPtr]::Zero,[IntPtr]::Zero)){throw 'WM_CLOSE failed'}
 if(-not $p.WaitForExit(15000)){throw 'Native installer did not exit after close'}
 if($p.ExitCode -ne 0){throw "Installer exit code: $($p.ExitCode)"}
} finally {
 if(-not $p.HasExited){$p.Kill();$p.WaitForExit()}
 Get-Content .tmp/installer-stdout.log,.tmp/installer-stderr.log -ErrorAction SilentlyContinue
}
