//go:build windows

package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
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

const version = "0.1.2"
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ConnectApp`

type setup struct {
	mu        sync.Mutex
	app       *application.App
	token     string
	uninstall bool
	removed   bool
}

func installDir() string    { return filepath.Join(os.Getenv("ProgramFiles"), "Connect") }
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
	dir := installDir()
	if !filepath.IsAbs(dir) || filepath.Base(dir) != "Connect" {
		return "", fmt.Errorf("invalid installation directory")
	}
	if st, err := os.Lstat(dir); err == nil && (st.Mode()&os.ModeSymlink != 0 || !st.IsDir()) {
		return "", fmt.Errorf("installation directory must be a regular folder")
	}
	return dir, nil
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
func (s *setup) install(desktop bool, chrome bool) error {
	dir, err := safeDir()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err = writeAtomic(filepath.Join(dir, "Connect.exe"), payload); err != nil {
		return fmt.Errorf("close Connect before updating: %w", err)
	}
	if chrome {
		if err = installChrome(dir); err != nil {
			return err
		}
	}
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
	if err = os.MkdirAll(menu, 0755); err != nil {
		return err
	}
	script := "$ErrorActionPreference='Stop'; $w=New-Object -ComObject WScript.Shell; "
	destinations := []string{filepath.Join(menu, "Connect.lnk")}
	if desktop {
		destinations = append(destinations, filepath.Join(os.Getenv("PUBLIC"), "Desktop", "Connect.lnk"))
	}
	for _, dest := range destinations {
		script += "$l=$w.CreateShortcut(" + quote(dest) + "); $l.TargetPath=" + quote(filepath.Join(dir, "Connect.exe")) + "; $l.WorkingDirectory=" + quote(dir) + "; $l.IconLocation=" + quote(filepath.Join(dir, "Connect.exe")+",0") + "; $l.Save(); "
	}
	if err = powershell(script); err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, uninstallKey, registry.SET_VALUE)
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
	return nil
}
func (s *setup) remove() error {
	dir, err := safeDir()
	if err != nil {
		return err
	}
	// Remove the app first. A running app leaves everything intact and reports an error.
	if err = os.Remove(filepath.Join(dir, "Connect.exe")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("close Connect before uninstalling: %w", err)
	}
	_ = registry.DeleteKey(registry.LOCAL_MACHINE, `Software\Google\Chrome\NativeMessagingHosts\com.sanderstripa.connect`)
	_ = os.Remove(filepath.Join(dir, "chrome-native-host.json"))
	if err = registry.DeleteKey(registry.LOCAL_MACHINE, uninstallKey); err != nil && err != registry.ErrNotExist {
		return err
	}
	menu := filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs\Connect`)
	_ = os.Remove(filepath.Join(menu, "Connect.lnk"))
	_ = os.Remove(menu)
	_ = os.Remove(filepath.Join(os.Getenv("PUBLIC"), "Desktop", "Connect.lnk"))
	script := "$ErrorActionPreference='Stop'; $id=[Security.Principal.WindowsIdentity]::GetCurrent(); $task='Connect-App-'+$id.User.Value; if(Get-ScheduledTask -TaskName $task -ErrorAction SilentlyContinue){Unregister-ScheduledTask -TaskName $task -Confirm:$false}; Remove-ItemProperty -LiteralPath 'HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Run' -Name 'connect-app' -ErrorAction SilentlyContinue"
	if err = powershell(script); err != nil {
		return err
	}
	return nil
}
func (s *setup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.Header.Get("X-Setup-Token") != s.token {
		http.Error(w, "forbidden", 403)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	switch r.URL.Path {
	case "/api/install":
		var opts struct {
			Desktop bool `json:"desktop"`
			Chrome  bool `json:"chrome"`
		}
		err = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&opts)
		if err == nil {
			err = s.install(opts.Desktop, opts.Chrome)
		}
	case "/api/uninstall":
		if !s.uninstall {
			err = fmt.Errorf("invalid mode")
		} else {
			err = s.remove()
			s.removed = err == nil
		}
	case "/api/open":
		// Explorer launches the desktop interface at the user's normal integrity
		// instead of inheriting the installer's administrator token.
		cmd := exec.Command("explorer.exe", filepath.Join(installDir(), "Connect.exe"))
		err = cmd.Start()
	case "/api/chrome":
		if s.uninstall {
			err = fmt.Errorf("invalid mode")
		} else {
			err = prepareChrome(installDir())
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
	s := &setup{token: hex.EncodeToString(tokenBytes)}
	for _, arg := range os.Args[1:] {
		if arg == "--uninstall" {
			s.uninstall = true
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
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, html)
			return
		}
		files.ServeHTTP(w, r)
	})
	s.app = application.New(application.Options{Name: "Connect Setup", Icon: icon, Assets: application.AssetOptions{Handler: handler}, SingleInstance: &application.SingleInstanceOptions{UniqueID: "com.sanderstripa.sudoku.setup", ExitCode: 0}})
	s.app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "Connect Setup", Width: 440, Height: 600, MinWidth: 440, MinHeight: 600, Frameless: true, BackgroundColour: application.NewRGB(12, 12, 13)})
	if err := s.app.Run(); err != nil {
		panic(err)
	}
	if s.uninstall && s.removed {
		dir, err := safeDir()
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
	key, _, e := registry.CreateKey(registry.LOCAL_MACHINE, `Software\Google\Chrome\NativeMessagingHosts\com.sanderstripa.connect`, registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer key.Close()
	return key.SetStringValue("", p)
}
