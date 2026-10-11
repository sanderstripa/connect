//go:build windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"syscall"
	"unsafe"
)

func trayAnchor() (application.Point, application.Rect, bool) {
	var p struct{ X, Y int32 }
	proc := syscall.NewLazyDLL("user32.dll").NewProc("GetCursorPos")
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(&p)))
	if ok == 0 {
		return application.Point{}, application.Rect{}, false
	}
	physical := application.Point{X: int(p.X), Y: int(p.Y)}
	screen := application.ScreenNearestPhysicalPoint(physical)
	if screen == nil {
		return application.Point{}, application.Rect{}, false
	}
	return application.PhysicalToDipPoint(physical), application.PhysicalToDipRect(screen.PhysicalWorkArea), true
}
