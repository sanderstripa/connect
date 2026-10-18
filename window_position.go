package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type windowPosition struct {
	X        int    `json:"x"`
	Y        int    `json:"y"`
	ScreenID string `json:"screenId"`
	OffsetX  int    `json:"offsetX"`
	OffsetY  int    `json:"offsetY"`
}

func windowPositionPath() string {
	root, e := os.UserConfigDir()
	if e != nil {
		return ""
	}
	return filepath.Join(root, "connect-app", "window-position.json")
}
func loadWindowPosition(file string) (windowPosition, bool) {
	var p windowPosition
	data, e := os.ReadFile(file)
	if e != nil || len(data) > 4096 || json.Unmarshal(data, &p) != nil {
		return p, false
	}
	for _, v := range []int{p.X, p.Y, p.OffsetX, p.OffsetY} {
		if v < -1000000 || v > 1000000 {
			return p, false
		}
	}
	return p, true
}
func saveWindowPosition(file string, p windowPosition) error {
	if file == "" {
		return nil
	}
	if e := os.MkdirAll(filepath.Dir(file), 0700); e != nil {
		return e
	}
	data, e := json.Marshal(p)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(file), ".window-position-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(data); e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, file)
}
func restoredWindowPosition(p windowPosition, screens []*application.Screen, width, height int) (int, int, bool) {
	var target *application.Screen
	distance := math.MaxFloat64
	for _, s := range screens {
		if s == nil || s.WorkArea.Width <= 0 || s.WorkArea.Height <= 0 {
			continue
		}
		if s.ID == p.ScreenID && p.ScreenID != "" {
			target = s
			break
		}
		r := s.WorkArea
		dx := math.Max(float64(r.X-p.X), math.Max(0, float64(p.X-r.X-r.Width)))
		dy := math.Max(float64(r.Y-p.Y), math.Max(0, float64(p.Y-r.Y-r.Height)))
		d := dx*dx + dy*dy
		if d < distance {
			distance = d
			target = s
		}
	}
	if target == nil {
		return 0, 0, false
	}
	r := target.WorkArea
	x, y := p.X, p.Y
	if target.ID == p.ScreenID && p.ScreenID != "" {
		x = r.X + p.OffsetX
		y = r.Y + p.OffsetY
	}
	x = max(r.X, min(x, r.X+max(0, r.Width-width)))
	y = max(r.Y, min(y, r.Y+max(0, r.Height-height)))
	return x, y, true
}

// Only the main UI owns this small file. It never rewrites VPN profiles or
// saves minimized/maximized coordinates. Movement is already debounced by Wails.
func bindWindowPosition(app *application.App, w application.Window, startHidden bool) func() {
	file := windowPositionPath()
	var ready atomic.Bool
	var mu sync.Mutex
	save := func() {
		if !ready.Load() || w.IsMinimised() || w.IsMaximised() {
			return
		}
		x, y := w.Position()
		p := windowPosition{X: x, Y: y}
		if s, e := w.GetScreen(); e == nil && s != nil {
			p.ScreenID = s.ID
			p.OffsetX = x - s.WorkArea.X
			p.OffsetY = y - s.WorkArea.Y
		}
		// Never hold the file lock across synchronous UI-thread calls: quitting
		// and the debounced movement callback can run concurrently.
		mu.Lock()
		defer mu.Unlock()
		_ = saveWindowPosition(file, p)
	}
	w.OnWindowEvent(events.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
		if p, ok := loadWindowPosition(file); ok {
			width, height := w.Size()
			if x, y, valid := restoredWindowPosition(p, app.Screen.GetAll(), width, height); valid {
				w.SetPosition(x, y)
			}
		}
		ready.Store(true)
		if !startHidden {
			w.Show()
		}
	})
	w.OnWindowEvent(events.Common.WindowDidMove, func(_ *application.WindowEvent) {
		if w.IsVisible() {
			save()
		}
	})
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) { save() })
	return save
}
