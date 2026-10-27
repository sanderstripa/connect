package main

import (
	"embed"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/core"
	"github.com/SUDOKU-ASCII/sudoku/pkg/logx"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed all:runtime/bin
var bundledRuntime embed.FS

//go:embed build/appicon.png
var trayIcon []byte

//go:embed build/tray/*.png
var trayImages embed.FS

//go:embed build/tray/logo-monochrome.svg
var trayTemplateIcon []byte

func init() {
	application.RegisterEvent[core.RuntimeState](core.EventStateUpdated)
	application.RegisterEvent[core.LogEntry](core.EventLogAdded)
}

func main() {
	if runChromeHost() {
		return
	}
	if runNetworkHelper() {
		return
	}
	startHidden := false
	for _, arg := range os.Args[1:] {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "--autostart", "--background", "--hidden":
			startHidden = true
		}
	}

	appService := NewApp(bundledRuntime, "runtime/bin")
	var quitting atomic.Bool
	var app *application.App
	var mainWindow application.Window
	var persistMainPosition func()

	showExistingInstance := func() {
		application.InvokeAsync(func() {
			if app != nil {
				app.Show()
			}
			if mainWindow != nil {
				if mainWindow.IsMinimised() {
					mainWindow.UnMinimise()
				}
				mainWindow.Restore()
				mainWindow.Show().Focus()
			}
			if app != nil {
				dialog := app.Dialog.Info().
					SetTitle("Connect уже запущен").
					SetMessage("Переключились на открытое окно Connect.")
				if mainWindow != nil {
					dialog.AttachToWindow(mainWindow)
				}
				dialog.Show()
			}
		})
	}

	app = application.New(application.Options{
		Name:        "connect-app",
		Description: "Минималистичный VPN-клиент Connect",
		Icon:        trayIcon,
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.sanderstripa.connect",
			ExitCode: 0,
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				showExistingInstance()
			},
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
			ProgramName:                   "connect-app",
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		ShouldQuit: func() bool {
			if persistMainPosition != nil {
				persistMainPosition()
			}
			quitting.Store(true)
			appService.ShutdownNow()
			return true
		},
	})

	requestQuit := func() {
		if !quitting.CompareAndSwap(false, true) {
			return
		}
		if persistMainPosition != nil {
			persistMainPosition()
		}
		appService.ShutdownNow()
		app.Quit()
	}

	backgroundType, backgroundColour := mainWindowBackground()
	mainWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                   "main",
		Title:                  "Connect",
		Width:                  410,
		Height:                 660,
		MinWidth:               360,
		MinHeight:              660,
		URL:                    "/",
		Frameless:              true,
		BackgroundType:         backgroundType,
		BackgroundColour:       backgroundColour,
		Hidden:                 true,
		OpenInspectorOnStartup: false,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent,
		},
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: false,
		},
		Linux: application.LinuxWindow{
			Icon:                trayIcon,
			WindowIsTranslucent: true,
		},
		KeyBindings: map[string]func(window application.Window){
			"cmd+q": func(window application.Window) {
				requestQuit()
			},
			"ctrl+q": func(window application.Window) {
				requestQuit()
			},
			"cmd+w": func(window application.Window) {
				if runtime.GOOS == "darwin" {
					window.Hide()
				}
			},
			"ctrl+w": func(window application.Window) {
				if runtime.GOOS == "darwin" {
					window.Hide()
				}
			},
		},
	})

	configureWindowIcon(mainWindow)
	persistMainPosition = bindWindowPosition(app, mainWindow, startHidden)
	mainWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		persistMainPosition()
		if quitting.Load() {
			return
		}
		event.Cancel()
		mainWindow.Hide()
	})
	configureApplicationMenu(app, mainWindow)

	trayWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "tray-menu", Title: "Connect — меню", Width: 220, Height: 196,
		URL: "/?tray=1", Frameless: true, Hidden: true, AlwaysOnTop: true,
		ContentProtectionEnabled: false, BackgroundColour: backgroundColour,
		Windows: application.WindowsWindow{DisableFramelessWindowDecorations: false, HiddenOnTaskbar: true},
	})
	trayWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		trayWindow.Hide()
	})
	trayWindow.OnWindowEvent(events.Common.WindowLostFocus, func(_ *application.WindowEvent) {
		trayWindow.Hide()
	})
	tray := app.SystemTray.New()
	trayOffLight, trayOn := loadTrayImages()
	trayOffDark := trayOffLight
	tray.SetTooltip("Connect")
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayTemplateIcon)
	} else {
		tray.SetIcon(trayOffLight)
		tray.SetDarkModeIcon(trayOffDark)
	}
	go func() {
		<-appService.ready
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		wasConnected := false
		for range ticker.C {
			if quitting.Load() {
				return
			}
			state := appService.GetState()
			connected := trayConnected(state)
			if connected == wasConnected {
				continue
			}
			wasConnected = connected
			application.InvokeAsync(func() {
				if quitting.Load() {
					return
				}
				updateWindowConnectionIcon(mainWindow, connected)
				if connected {
					tray.SetIcon(trayOn).SetDarkModeIcon(trayOn).SetTooltip("Connect — подключено")
				} else {
					tray.SetIcon(trayOffLight).SetDarkModeIcon(trayOffDark).SetTooltip("Connect — отключено")
				}
			})
		}
	}()
	var menuAnchor application.Point
	var menuArea application.Rect
	var anchored bool
	positionTray := func() {
		if anchored {
			bounds := trayWindow.Bounds()
			p := trayMenuPosition(menuAnchor, menuArea, bounds.Width, bounds.Height)
			trayWindow.SetPosition(p.X, p.Y)
		} else {
			_ = tray.PositionWindow(trayWindow, 8)
		}
	}
	trayWindow.OnWindowEvent(events.Common.WindowDidResize, func(_ *application.WindowEvent) {
		if trayWindow.IsVisible() {
			positionTray()
		}
	})
	showTray := func() {
		if trayWindow.IsVisible() {
			trayWindow.Hide()
			return
		}
		menuAnchor, menuArea, anchored = trayAnchor()
		positionTray()
		trayWindow.Show().Focus()
	}
	tray.OnClick(func() {
		trayWindow.Hide()
		if mainWindow.IsMinimised() {
			mainWindow.UnMinimise()
		}
		mainWindow.Restore()
		mainWindow.Show().Focus()
	})
	tray.OnRightClick(showTray)
	app.Event.On("tray:open", func(event *application.CustomEvent) {
		page, _ := event.Data.(string)
		if page != "settings" && page != "servers" {
			page = "home"
		}
		application.InvokeAsync(func() {
			trayWindow.Hide()
			mainWindow.Show().Focus()
			app.Event.Emit("main:navigate", page)
		})
	})
	app.Event.On("tray:quit", func(_ *application.CustomEvent) { go requestQuit() })
	err := app.Run()
	if err != nil {
		logx.Errorf("Desktop", "wails run failed: %v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
