package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"os"
	"path/filepath"
	"testing"
)

type positionTestWindow struct {
	application.Window
	x, y                          int
	visible, minimized, maximized bool
	callbacks                     map[events.WindowEventType]func(*application.WindowEvent)
}

func (w *positionTestWindow) OnWindowEvent(e events.WindowEventType, cb func(*application.WindowEvent)) func() {
	w.callbacks[e] = cb
	return func() {}
}
func (w *positionTestWindow) Position() (int, int) { return w.x, w.y }
func (w *positionTestWindow) GetScreen() (*application.Screen, error) {
	return &application.Screen{ID: "main", WorkArea: application.Rect{Width: 1920, Height: 1040}}, nil
}
func (w *positionTestWindow) IsVisible() bool          { return w.visible }
func (w *positionTestWindow) IsMinimised() bool        { return w.minimized }
func (w *positionTestWindow) IsMaximised() bool        { return w.maximized }
func (w *positionTestWindow) Show() application.Window { w.visible = true; return w }

func TestWindowPositionEvents(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "hidden-autostart"}[hidden], func(t *testing.T) {
			t.Setenv("APPDATA", t.TempDir())
			w := &positionTestWindow{x: 321, y: 123, callbacks: map[events.WindowEventType]func(*application.WindowEvent){}}
			persist := bindWindowPosition(nil, w, hidden)
			persist()
			if _, ok := loadWindowPosition(windowPositionPath()); ok {
				t.Fatal("saved before runtime ready")
			}
			w.callbacks[events.Common.WindowRuntimeReady](nil)
			if w.visible == hidden {
				t.Fatal("autostart visibility changed")
			}
			w.visible = true
			w.callbacks[events.Common.WindowDidMove](nil)
			p, ok := loadWindowPosition(windowPositionPath())
			if !ok || p.X != 321 || p.Y != 123 {
				t.Fatalf("move not saved: %+v", p)
			}
			w.minimized = true
			w.x = -32000
			persist()
			w.minimized = false
			w.maximized = true
			persist()
			w.maximized = false
			p, _ = loadWindowPosition(windowPositionPath())
			if p.X != 321 {
				t.Fatal("non-normal coordinates replaced saved location")
			}
			w.x = 620
			w.y = 210
			w.callbacks[events.Common.WindowClosing](nil)
			p, _ = loadWindowPosition(windowPositionPath())
			if p.X != 620 || p.Y != 210 {
				t.Fatal("closing did not capture final position")
			}
		})
	}
}

func TestWindowPositionPersistence(t *testing.T) {
	file := filepath.Join(t.TempDir(), "ui", "window-position.json")
	if _, ok := loadWindowPosition(file); ok {
		t.Fatal("missing position accepted")
	}
	for _, p := range []windowPosition{{X: 500, Y: 120, ScreenID: "main", OffsetX: 500, OffsetY: 120}, {X: -1450, Y: 75, ScreenID: "left", OffsetX: 470, OffsetY: 75}} {
		if err := saveWindowPosition(file, p); err != nil {
			t.Fatal(err)
		}
		got, ok := loadWindowPosition(file)
		if !ok || got != p {
			t.Fatalf("restart lost position: %+v %v", got, ok)
		}
	}
	for _, bad := range []string{"broken", `{"x":-3200000}`, string(make([]byte, 4097))} {
		if err := os.WriteFile(file, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if _, ok := loadWindowPosition(file); ok {
			t.Fatal("invalid position accepted")
		}
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(file), ".window-position-*"))
	if len(files) != 0 {
		t.Fatal("temporary files leaked")
	}
}

func TestWindowPositionMonitorRecovery(t *testing.T) {
	screen := func(id string, x, y, w, h int) *application.Screen {
		return &application.Screen{ID: id, WorkArea: application.Rect{X: x, Y: y, Width: w, Height: h}}
	}
	primary := screen("main", 0, 0, 1920, 1040)
	left := screen("left", -1920, 0, 1920, 1040)
	for _, tc := range []struct {
		name string
		p    windowPosition
		s    []*application.Screen
		x, y int
		ok   bool
	}{
		{"same monitor", windowPosition{X: 400, Y: 100, ScreenID: "main", OffsetX: 400, OffsetY: 100}, []*application.Screen{primary}, 400, 100, true},
		{"negative coordinates", windowPosition{X: -1500, Y: 120, ScreenID: "left", OffsetX: 420, OffsetY: 120}, []*application.Screen{primary, left}, -1500, 120, true},
		{"monitor moved", windowPosition{X: -1500, Y: 120, ScreenID: "left", OffsetX: 420, OffsetY: 120}, []*application.Screen{screen("left", 1920, 40, 1920, 1040)}, 2340, 160, true},
		{"removed monitor", windowPosition{X: -1500, Y: 120, ScreenID: "left", OffsetX: 420, OffsetY: 120}, []*application.Screen{primary}, 0, 120, true},
		{"clamp bottom right", windowPosition{X: 1900, Y: 1000}, []*application.Screen{primary}, 1510, 380, true},
		{"small work area", windowPosition{X: 600, Y: 1000}, []*application.Screen{screen("tiny", 100, 100, 320, 480)}, 100, 100, true},
		{"no monitors", windowPosition{}, nil, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x, y, ok := restoredWindowPosition(tc.p, tc.s, 410, 660)
			if x != tc.x || y != tc.y || ok != tc.ok {
				t.Fatalf("got %d,%d,%v want %d,%d,%v", x, y, ok, tc.x, tc.y, tc.ok)
			}
		})
	}
}
