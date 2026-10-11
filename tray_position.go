package main

import "github.com/wailsapp/wails/v3/pkg/application"

func trayMenuPosition(anchor application.Point, area application.Rect, width, height int) application.Point {
	const gap = 24
	x := anchor.X - width/2
	y := anchor.Y - height - gap
	if y < area.Y {
		y = anchor.Y + gap
	}
	x = max(area.X, min(x, area.X+area.Width-width))
	y = max(area.Y, min(y, area.Y+area.Height-height))
	return application.Point{X: x, Y: y}
}
