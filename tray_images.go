package main

import (
	"fmt"
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/core"
)

// A requested connection or a running core alone must never light up the tray.
func trayConnected(state core.RuntimeState) bool {
	return state.Running && state.CoreRunning && state.TunRunning && state.RouteSetupError == ""
}

func loadTrayImages() ([]byte, []byte) {
	size := trayImageSize()
	selected, distance := 32, 1000
	for _, candidate := range []int{16, 20, 24, 28, 32, 36, 40, 48, 64} {
		delta := candidate - size
		if delta < 0 {
			delta = -delta
		}
		if delta < distance {
			selected, distance = candidate, delta
		}
	}
	off, err := trayImages.ReadFile(fmt.Sprintf("build/tray/off-%d.png", selected))
	if err != nil {
		panic(err)
	}
	on, err := trayImages.ReadFile(fmt.Sprintf("build/tray/on-%d.png", selected))
	if err != nil {
		panic(err)
	}
	return off, on
}
