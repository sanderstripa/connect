//go:build windows

package core

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.org/x/sys/windows"
)

//go:embed web
var assets embed.FS

type Host struct {
	config       Config
	adapter      Adapter
	token        string
	app          *application.App
	mu           sync.Mutex
	uninstall    bool
	installed    bool
	removed      bool
	chooseFolder func() (string, error)
}

func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		if r.URL.Path == "/assets/mobius.png" && h.config.Ribbon != nil {
			data, err := fs.ReadFile(h.config.Ribbon, "ribbon.png")
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(data)
			return
		}
		if r.URL.Path == "/" {
			raw, _ := assets.ReadFile("web/index.html")
			cfg, _ := json.Marshal(h.config)
			html := strings.NewReplacer("__TOKEN__", h.token, "__MODE__", fmt.Sprint(h.uninstall), "__CONFIG__", strings.ReplaceAll(string(cfg), "<", `\u003c`)).Replace(string(raw))
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, html)
			return
		}
		root, _ := fs.Sub(assets, "web")
		http.FileServer(http.FS(root)).ServeHTTP(w, r)
		return
	}
	if r.Header.Get("X-Setup-Token") != h.token || (r.Method != "POST" && !(r.Method == "GET" && r.URL.Path == "/api/progress")) {
		http.Error(w, "forbidden", 403)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/api/progress" {
		json.NewEncoder(w).Encode(h.adapter.Progress())
		return
	}
	if !h.mu.TryLock() {
		http.Error(w, "operation in progress", 409)
		return
	}
	defer h.mu.Unlock()
	var result any = map[string]bool{"ok": true}
	var err error
	switch r.URL.Path {
	case "/api/install":
		var opts Options
		err = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&opts)
		if h.uninstall {
			err = fmt.Errorf("invalid mode")
		}
		if err == nil && (opts.Language != "ru" && opts.Language != "en" || opts.Theme != "light" && opts.Theme != "dark") {
			err = fmt.Errorf("invalid preferences")
		}
		if err == nil {
			err = h.adapter.Install(opts)
			h.installed = err == nil
		}
	case "/api/uninstall":
		if !h.uninstall {
			err = fmt.Errorf("invalid mode")
		} else {
			err = h.adapter.Remove()
			h.removed = err == nil
		}
	case "/api/open":
		if h.uninstall || !h.installed {
			err = fmt.Errorf("installation not completed")
		} else {
			err = h.adapter.Open()
		}
	case "/api/folder":
		if h.uninstall {
			err = fmt.Errorf("invalid mode")
			break
		}
		var p string
		p, err = h.chooseFolder()
		var code interface{ Code() uintptr }
		cancelled := errors.Is(err, windows.ERROR_CANCELLED) || (errors.As(err, &code) && uint32(code.Code()) == 0x800704c7)
		if err != nil && (cancelled || strings.Contains(err.Error(), "cancelled by user") || strings.Contains(err.Error(), "0x800704c7")) {
			err = nil
			p = ""
		}
		result = map[string]any{"path": p, "cancelled": p == ""}
	case "/api/close":
		go h.app.Quit()
	default:
		if !h.installed || h.uninstall {
			err = fmt.Errorf("installation not completed")
			break
		}
		var body []byte
		body, err = io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
		if err == nil {
			result, err = h.adapter.Action(strings.TrimPrefix(r.URL.Path, "/api/"), body)
		}
	}
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}

func Run(config Config, adapter Adapter) error {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return err
	}
	h := &Host{config: config, adapter: adapter, token: hex.EncodeToString(token)}
	for _, arg := range os.Args[1:] {
		if arg == "--uninstall" {
			h.uninstall = true
		}
	}
	icon, _ := assets.ReadFile("web/assets/icon.png")
	if len(config.Icon) > 0 {
		icon = config.Icon
	}
	h.app = application.New(application.Options{Name: config.Name + " Setup", Icon: icon, Windows: application.WindowsOptions{WebviewUserDataPath: os.Getenv("WEBVIEW2_USER_DATA_FOLDER"), AdditionalBrowserArgs: strings.Fields(os.Getenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS"))}, Assets: application.AssetOptions{Handler: h}, SingleInstance: &application.SingleInstanceOptions{UniqueID: "com.sanderstripa.installer." + config.ID, ExitCode: 0}})
	window := h.app.Window.NewWithOptions(application.WebviewWindowOptions{Title: config.Name, Width: 840, Height: 540, MinWidth: 840, MinHeight: 540, MaxWidth: 840, MaxHeight: 540, DisableResize: true, Frameless: true, BackgroundColour: application.NewRGB(242, 240, 235)})
	h.chooseFolder = func() (string, error) {
		return h.app.Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false).PromptForSingleSelection()
	}
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !h.mu.TryLock() {
			event.Cancel()
			return
		}
		h.mu.Unlock()
	})
	err := h.app.Run()
	if err == nil && h.removed {
		if cleaner, ok := adapter.(interface{ AfterRemoval() error }); ok {
			return cleaner.AfterRemoval()
		}
	}
	return err
}
