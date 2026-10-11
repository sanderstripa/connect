//go:build windows

package main

import "github.com/wailsapp/wails/v3/pkg/w32"

func trayImageSize() int {
	return int(w32.GetSystemMetrics(w32.SM_CXSMICON))
}
