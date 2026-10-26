//go:build windows

package vpn

import (
	"golang.org/x/sys/windows"
	"runtime"
	"unsafe"
)

// Notify Explorer after replacing the executable at the same path on upgrade.
// Do not delete the user's icon cache or restart their shell.
func refreshShellIcons(paths ...string) {
	notify := windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")
	if notify.Find() != nil {
		return
	}
	for _, path := range paths {
		name, err := windows.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		// SHCNE_UPDATEITEM, SHCNF_PATHW | SHCNF_FLUSHNOWAIT.
		notify.Call(0x2000, 0x2005, uintptr(unsafe.Pointer(name)), 0)
		runtime.KeepAlive(name)
	}
}
