//go:build windows

package main

import (
 "embed"
 "fmt"
 "runtime"
 "unsafe"

 "github.com/wailsapp/wails/v3/pkg/application"
 "github.com/wailsapp/wails/v3/pkg/events"
 "github.com/wailsapp/wails/v3/pkg/w32"
 "golang.org/x/sys/windows"
)

//go:embed build/branding/icon-*.png build/branding/app-on-*.png
var windowStateImages embed.FS

// Accessed only on the application's UI thread. Keep native icon handles alive
// for the process lifetime: Windows retains them after WM_SETICON, including
// when a window is hidden and restored. The cache is bounded by DPI and state.
var windowIconConnected bool
var stateIconHandles = map[string]uintptr{}
var createStateIcon = windows.NewLazySystemDLL("user32.dll").NewProc("CreateIconFromResourceEx")

func applyWindowStateIcon(window application.Window) {
 hwnd := w32.HWND(uintptr(window.NativeWindow()))
 if hwnd == 0 { return }
 for _, icon := range []struct{ kind uintptr; width, height int }{
  {w32.ICON_SMALL, w32.GetSystemMetrics(w32.SM_CXSMICON), w32.GetSystemMetrics(w32.SM_CYSMICON)},
  {w32.ICON_BIG, w32.GetSystemMetrics(w32.SM_CXICON), w32.GetSystemMetrics(w32.SM_CYICON)},
 } {
  size, distance := 32, 10000
  for _, n := range []int{16,20,24,28,32,36,40,48,64,96,128,256} {
   d := n-icon.width; if d<0 { d = -d }; if d<distance { size,distance=n,d }
  }
  prefix := "icon"; if windowIconConnected { prefix="app-on" }
  key := fmt.Sprintf("%s-%d-%d-%d",prefix,size,icon.width,icon.height)
  handle := stateIconHandles[key]
  if handle==0 {
   png,err := windowStateImages.ReadFile(fmt.Sprintf("build/branding/%s-%d.png",prefix,size))
   if err!=nil || len(png)==0 { continue }
   // PNG-compressed RT_ICON payloads are supported by Windows 11.
   handle,_,_ = createStateIcon.Call(uintptr(unsafe.Pointer(&png[0])),uintptr(len(png)),1,0x00030000,uintptr(icon.width),uintptr(icon.height),0)
   runtime.KeepAlive(png)
   if handle==0 { continue }
   stateIconHandles[key]=handle
  }
  w32.SendMessage(hwnd,w32.WM_SETICON,icon.kind,handle)
 }
}

func configureWindowIcon(window application.Window) {
 apply := func(_ *application.WindowEvent) { application.InvokeAsync(func(){applyWindowStateIcon(window)}) }
 window.OnWindowEvent(events.Common.WindowRuntimeReady,apply)
 window.OnWindowEvent(events.Common.WindowDPIChanged,apply)
}

func updateWindowConnectionIcon(window application.Window, connected bool) {
 windowIconConnected=connected
 applyWindowStateIcon(window)
}
