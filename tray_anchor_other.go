//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func trayAnchor() (application.Point, application.Rect, bool) {
	return application.Point{}, application.Rect{}, false
}
