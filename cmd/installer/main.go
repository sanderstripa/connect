//go:build windows

package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/installprefs"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed web
var web embed.FS

//go:embed payload/Connect.exe
var payload []byte

//go:embed chrome
var chromeFiles embed.FS

//go:embed web/assets/icon.png
var icon []byte

var version = "0.2.6" // Overridden from the application package by build-installer.ps1.
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ConnectApp`

var machineRegistry = registry.LOCAL_MACHINE

type setup struct {
	mu               sync.Mutex
	app              *application.App
	token            string
	uninstall        bool
	removed          bool
	dir              string
	installed        bool
	chrome           bool
	progress         atomic.Int32
	cleanupAutostart func() error
	chooseFolder     func() (string, error)
}

func installDir() string {
	key, err := registry.OpenKey(machineRegistry, uninstallKey, registry.QUERY_VALUE)
	if err == nil {
		defer key.Close()
		if dir, _, e := key.GetStringValue("InstallLocation"); e == nil && filepath.IsAbs(dir) {
			return filepath.Clean(dir)
		}
	}
	return filepath.Join(os.Getenv("ProgramFiles"), "Connect")
}
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func powershell(script string) error {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
func safeDir() (string, error) {
	return validateDir(installDir())
}
func validateDir(dir string) (string, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if !filepath.IsAbs(dir) || strings.HasPrefix(dir, `\\`) || filepath.Dir(dir) == dir || strings.ContainsAny(dir, `<>"|?*`) {
		return "", fmt.Errorf("invalid installation directory")
	}
	if strings.Contains(strings.TrimPrefix(dir, filepath.VolumeName(dir)), ":") {
		return "", fmt.Errorf("invalid installation directory")
	}
	for _, protected := range []string{os.Getenv("WINDIR"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramData"), os.Getenv("USERPROFILE")} {
		if protected != "" && strings.EqualFold(dir, filepath.Clean(protected)) {
			return "", fmt.Errorf("choose a dedicated application folder")
		}
	}
	for p := dir; filepath.Dir(p) != p; p = filepath.Dir(p) {
		if st, err := os.Lstat(p); err == nil {
			attrs, e := windows.GetFileAttributes(windows.StringToUTF16Ptr(p))
			if e != nil || !st.IsDir() || attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
				return "", fmt.Errorf("installation directory must not contain junctions or symbolic links")
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return dir, nil
}

// Opening with DELETE access fails while the executable is mapped by Windows.
// Check before changing extension files, shortcuts or registry. Never kill a process.
func checkAppClosed(dir string) error {
	p := filepath.Join(dir, "Connect.exe")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	h, err := windows.CreateFile(windows.StringToUTF16Ptr(p), windows.DELETE|windows.GENERIC_WRITE, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return fmt.Errorf("close Connect from its tray menu, disconnect the Chrome extension, then retry: %w", err)
	}
	return windows.CloseHandle(h)
}
func writeAtomic(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".sudoku-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func (s *setup) install(desktop bool, chrome bool, startMenu bool, destination string, preferences ...installprefs.Preferences) error {
	var preferenceData []byte
	if len(preferences) > 0 {
		p := preferences[0]
		id := make([]byte, 32)
		if _, err := rand.Read(id); err != nil {
			return err
		}
		p.ID = hex.EncodeToString(id)
		if !p.Valid() {
			return fmt.Errorf("invalid installation language or theme")
		}
		preferenceData, _ = json.Marshal(p)
	}
	s.progress.Store(0)
	dir, err := validateDir(destination)
	if err != nil {
		return err
	}
	if err = checkAppClosed(dir); err != nil {
		return err
	}
	// A custom destination must be empty or already owned by CONNECT.
	if entries, e := os.ReadDir(dir); e == nil && len(entries) > 0 {
		if _, e = os.Stat(filepath.Join(dir, "Connect.exe")); e != nil {
			return fmt.Errorf("choose an empty folder or the existing CONNECT installation")
		}
	}
	s.progress.Store(5)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err = writeAtomic(filepath.Join(dir, "Connect.exe"), payload); err != nil {
		return fmt.Errorf("close Connect before updating: %w", err)
	}
	s.progress.Store(45)
	if chrome {
		if err = installChrome(dir); err != nil {
			return err
		}
	}
	s.progress.Store(65)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	if err = writeAtomic(filepath.Join(dir, "Uninstall.exe"), data); err != nil {
		return err
	}
	menu := filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs\Connect`)
	if startMenu {
		if err = os.MkdirAll(menu, 0755); err != nil {
			return err
		}
	}
	script := "$ErrorActionPreference='Stop'; $w=New-Object -ComObject WScript.Shell; "
	destinations := []string{}
	if startMenu {
		destinations = append(destinations, filepath.Join(menu, "Connect.lnk"))
	}
	if desktop {
		destinations = append(destinations, filepath.Join(os.Getenv("PUBLIC"), "Desktop", "Connect.lnk"))
	}
	for _, dest := range destinations {
		script += "$l=$w.CreateShortcut(" + quote(dest) + "); $l.TargetPath=" + quote(filepath.Join(dir, "Connect.exe")) + "; $l.WorkingDirectory=" + quote(dir) + "; $l.IconLocation=" + quote(filepath.Join(dir, "Connect.exe")+",0") + "; $l.Save(); "
	}
	if err = powershell(script); err != nil {
		return err
	}
	s.progress.Store(85)
	key, _, err := registry.CreateKey(machineRegistry, uninstallKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	values := map[string]string{"DisplayName": "Connect VPN", "DisplayVersion": version, "Publisher": "Connect", "InstallLocation": dir, "DisplayIcon": filepath.Join(dir, "Connect.exe"), "UninstallString": "\"" + filepath.Join(dir, "Uninstall.exe") + "\" --uninstall"}
	for k, v := range values {
		if err = key.SetStringValue(k, v); err != nil {
			return err
		}
	}
	for _, k := range []string{"NoModify", "NoRepair"} {
		if err = key.SetDWordValue(k, 1); err != nil {
			return err
		}
	}
	if preferenceData != nil {
		if err = writeAtomic(filepath.Join(dir, installprefs.File), preferenceData); err != nil {
			return err
		}
	}
	s.dir, s.installed, s.chrome = dir, true, chrome
	s.progress.Store(100)
	return nil
}
func (s *setup) remove() error {
	dir, err := validateDir(s.dir)
	if err != nil {
		return err
	}
	if err = checkAppClosed(dir); err != nil {
		return err
	}
	s.progress.Store(10)
	// Remove the app first. A running app leaves everything intact and reports an error.
	if err = os.Remove(filepath.Join(dir, "Connect.exe")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("close Connect before uninstalling: %w", err)
	}
	if err = os.Remove(filepath.Join(dir, installprefs.File)); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.progress.Store(50)
	_ = registry.DeleteKey(machineRegistry, `Software\Google\Chrome\NativeMessagingHosts\com.sanderstripa.connect`)
	_ = os.Remove(filepath.Join(dir, "chrome-native-host.json"))
	if err = registry.DeleteKey(machineRegistry, uninstallKey); err != nil && err != registry.ErrNotExist {
		return err
	}
	menu := filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs\Connect`)
	_ = os.Remove(filepath.Join(menu, "Connect.lnk"))
	_ = os.Remove(menu)
	_ = os.Remove(filepath.Join(os.Getenv("PUBLIC"), "Desktop", "Connect.lnk"))
	cleanup := s.cleanupAutostart
	if cleanup == nil {
		cleanup = cleanupAutostart
	}
	if err = cleanup(); err != nil {
		return err
	}
	s.progress.Store(100)
	return nil
}
func cleanupAutostart() error {
	return powershell("$ErrorActionPreference='Stop'; $id=[Security.Principal.WindowsIdentity]::GetCurrent(); $task='Connect-App-'+$id.User.Value; if(Get-ScheduledTask -TaskName $task -ErrorAction SilentlyContinue){Unregister-ScheduledTask -TaskName $task -Confirm:$false}; Remove-ItemProperty -LiteralPath 'HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Run' -Name 'connect-app' -ErrorAction SilentlyContinue")
}
func (s *setup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if (r.Method != "POST" && !(r.Method == "GET" && r.URL.Path == "/api/progress")) || r.Header.Get("X-Setup-Token") != s.token {
		http.Error(w, "forbidden", 403)
		return
	}
	if r.URL.Path == "/api/progress" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int32{"progress": s.progress.Load()})
		return
	}
	if !s.mu.TryLock() {
		http.Error(w, "operation in progress", http.StatusConflict)
		return
	}
	defer s.mu.Unlock()
	var err error
	switch r.URL.Path {
	case "/api/install":
		var opts struct {
			Desktop   bool   `json:"desktop"`
			Chrome    bool   `json:"chrome"`
			StartMenu *bool  `json:"startMenu"`
			Path      string `json:"path"`
			Language  string `json:"language"`
			Theme     string `json:"theme"`
		}
		err = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&opts)
		if s.uninstall {
			err = fmt.Errorf("invalid mode")
		}
		if err == nil {
			if opts.Path == "" {
				opts.Path = s.dir
			}
			// Existing installations stay at their registered location when updated.
			if _, e := os.Stat(filepath.Join(s.dir, "Connect.exe")); e == nil && !strings.EqualFold(filepath.Clean(opts.Path), s.dir) {
				err = fmt.Errorf("update the existing installation at %s", s.dir)
			} else {
				err = s.install(opts.Desktop, opts.Chrome, opts.StartMenu == nil || *opts.StartMenu, opts.Path, installprefs.Preferences{Language: opts.Language, Theme: opts.Theme})
			}
		}
	case "/api/folder":
		if s.uninstall {
			err = fmt.Errorf("invalid mode")
			break
		}
		choose := s.chooseFolder
		if choose == nil {
			choose = func() (string, error) {
				return s.app.Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false).PromptForSingleSelection()
			}
		}
		dir, e := choose()
		selection, e := folderDialogResult(dir, e)
		if e != nil {
			err = e
			break
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(selection)
		return
	case "/api/uninstall":
		if !s.uninstall {
			err = fmt.Errorf("invalid mode")
		} else {
			err = s.remove()
			s.removed = err == nil
		}
	case "/api/open":
		if !s.installed || s.uninstall {
			err = fmt.Errorf("installation not completed")
			break
		}
		// Explorer launches the desktop interface at the user's normal integrity
		// instead of inheriting the installer's administrator token.
		cmd := exec.Command("explorer.exe", filepath.Join(s.dir, "Connect.exe"))
		err = cmd.Start()
	case "/api/chrome":
		if s.uninstall || !s.installed || !s.chrome {
			err = fmt.Errorf("invalid mode")
		} else {
			err = prepareChrome(s.dir)
		}
	case "/api/close":
		go s.app.Quit()
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// Match the supplied installer's guided final step without modifying Chrome's
// authenticated profile files or imposing enterprise policies on a personal browser.
func prepareChrome(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "ChromeExtension", "manifest.json")); err != nil {
		return err
	}
	script := "$ErrorActionPreference='Stop'; Set-Clipboard -Value " + quote(filepath.Join(dir, "ChromeExtension")) + "; " +
		"$chrome=@((Join-Path $env:ProgramFiles 'Google\\Chrome\\Application\\chrome.exe'),(Join-Path ${env:ProgramFiles(x86)} 'Google\\Chrome\\Application\\chrome.exe'),(Join-Path $env:LOCALAPPDATA 'Google\\Chrome\\Application\\chrome.exe')) | Where-Object {Test-Path -LiteralPath $_} | Select-Object -First 1; " +
		"if($chrome){(New-Object -ComObject Shell.Application).ShellExecute($chrome,'chrome://extensions/','','open',1)}"
	return powershell(script)
}
func main() {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		panic(err)
	}
	s := &setup{token: hex.EncodeToString(tokenBytes), dir: installDir()}
	for _, arg := range os.Args[1:] {
		if arg == "--uninstall" {
			s.uninstall = true
		}
	}
	if s.uninstall {
		// The copied uninstaller owns its folder, including custom destinations.
		self, err := os.Executable()
		if err != nil {
			panic(err)
		}
		if strings.EqualFold(filepath.Base(self), "Uninstall.exe") {
			s.dir = filepath.Dir(self)
		}
		if !strings.EqualFold(filepath.Clean(s.dir), filepath.Clean(installDir())) {
			// Allow retrying cleanup after a previous removal deleted registration.
			if _, err := os.Stat(filepath.Join(s.dir, "Connect.exe")); !os.IsNotExist(err) || !strings.EqualFold(filepath.Base(self), "Uninstall.exe") {
				panic("uninstaller is not in the registered installation folder")
			}
		}
	}
	assets, _ := fs.Sub(web, "web")
	files := http.FileServer(http.FS(assets))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			s.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/" {
			data, _ := web.ReadFile("web/index.html")
			html := strings.ReplaceAll(string(data), "__TOKEN__", s.token)
			html = strings.ReplaceAll(html, "__MODE__", fmt.Sprint(s.uninstall))
			config, _ := json.Marshal(map[string]string{"path": s.dir, "version": version})
			html = strings.ReplaceAll(html, "__CONFIG__", strings.ReplaceAll(string(config), "<", `\u003c`))
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, html)
			return
		}
		files.ServeHTTP(w, r)
	})
	s.app = application.New(application.Options{Name: "Connect Setup", Icon: icon, Assets: application.AssetOptions{Handler: handler}, SingleInstance: &application.SingleInstanceOptions{UniqueID: "com.sanderstripa.sudoku.setup", ExitCode: 0}})
	window := s.app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "CONNECT VPN", Width: 840, Height: 540, MinWidth: 840, MinHeight: 540, MaxWidth: 840, MaxHeight: 540, DisableResize: true, Frameless: true, BackgroundColour: application.NewRGB(242, 240, 235)})
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !s.mu.TryLock() {
			event.Cancel()
			return
		}
		s.mu.Unlock()
	})
	if err := s.app.Run(); err != nil {
		panic(err)
	}
	if s.uninstall && s.removed {
		dir, err := validateDir(s.dir)
		if err == nil {
			// Delayed cleanup only removes our uninstaller and then the empty app folder.
			script := "Start-Sleep -Seconds 2; Remove-Item -LiteralPath " + quote(filepath.Join(dir, "Uninstall.exe")) + " -Force -ErrorAction SilentlyContinue; if((Get-ChildItem -LiteralPath " + quote(dir) + " -Force -ErrorAction SilentlyContinue | Measure-Object).Count -eq 0){Remove-Item -LiteralPath " + quote(dir) + " -ErrorAction SilentlyContinue}"
			// Cleanup is allowed only after the user completed removal.
			if _, err := os.Stat(filepath.Join(dir, "Connect.exe")); os.IsNotExist(err) {
				cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				_ = cmd.Start()
			}
		}
	}
}

func installChrome(dir string) error {
	root := filepath.Join(dir, "ChromeExtension")
	if err := fs.WalkDir(chromeFiles, "chrome", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		dest := filepath.Join(root, strings.TrimPrefix(strings.TrimPrefix(p, "chrome"), "/"))
		if d.IsDir() {
			if _, err := validateDir(dest); err != nil {
				return err
			}
			return os.MkdirAll(dest, 0755)
		}
		b, e := chromeFiles.ReadFile(p)
		if e != nil {
			return e
		}
		return writeAtomic(dest, b)
	}); err != nil {
		return err
	}
	manifest := map[string]any{"name": "com.sanderstripa.connect", "description": "Connect browser transport helper", "path": filepath.Join(dir, "Connect.exe"), "type": "stdio", "allowed_origins": []string{"chrome-extension://gmhibjohmhkchkongeibochdfidjiiic/"}}
	b, e := json.MarshalIndent(manifest, "", "  ")
	if e != nil {
		return e
	}
	p := filepath.Join(dir, "chrome-native-host.json")
	if e = writeAtomic(p, b); e != nil {
		return e
	}
	key, _, e := registry.CreateKey(machineRegistry, `Software\Google\Chrome\NativeMessagingHosts\com.sanderstripa.connect`, registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer key.Close()
	return key.SetStringValue("", p)
}
