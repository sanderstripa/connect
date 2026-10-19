//go:build windows

package screenshot

import (
	"connect-installer.local/core/core"
	"connect-installer.local/core/products"
	"embed"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

//go:embed Screenshot.exe ribbon.png
var assets embed.FS

const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Screenshot`
const settingsKey = `Software\SanderStripa\Screenshot`

func (a *Adapter) AfterRemoval() error { return core.CleanupUninstaller(a.dir, "Uninstall.exe") }

var user32 = windows.NewLazySystemDLL("user32.dll")

type Adapter struct {
	dir        string
	progress   atomic.Int32
	installed  bool
	hotkeyDone bool
}

func New() *Adapter {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Screenshot")
	if k, e := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE); e == nil {
		if p, _, e := k.GetStringValue("InstallLocation"); e == nil {
			dir = p
		}
		k.Close()
	}
	return &Adapter{dir: dir}
}
func (a *Adapter) Config() core.Config {
	return products.Configure("screenshot", a.dir, assets)
}
func (a *Adapter) Progress() core.Progress { return core.Progress{Percent: int(a.progress.Load())} }
func (a *Adapter) stop() error {
	// Match the application's cooperative exit event. Do not add process killing.
	event, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, windows.StringToUTF16Ptr(`Local\Screenshot.SanderStripa.Exit`))
	if err == nil {
		windows.SetEvent(event)
		windows.CloseHandle(event)
	}
	for i := 0; i < 30; i++ {
		if e := core.CheckClosed(filepath.Join(a.dir, "Screenshot.exe")); e == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("close Screenshot before continuing")
}
func writeReg(path string, values map[string]string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	for n, v := range values {
		if err = k.SetStringValue(n, v); err != nil {
			return err
		}
	}
	return nil
}
func (a *Adapter) Install(o core.Options) error {
	a.progress.Store(0)
	dir, err := core.ValidateDir(o.Path)
	if err != nil {
		return err
	}
	if _, e := os.Stat(filepath.Join(a.dir, "Screenshot.exe")); e == nil && !strings.EqualFold(dir, a.dir) {
		return fmt.Errorf("update the existing installation at %s", a.dir)
	}
	if entries, e := os.ReadDir(dir); e == nil && len(entries) > 0 {
		if _, e := os.Stat(filepath.Join(dir, "Screenshot.exe")); e != nil {
			return fmt.Errorf("choose an empty folder or existing Screenshot installation")
		}
	}
	existing := false
	if k, e := registry.OpenKey(registry.CURRENT_USER, settingsKey, registry.QUERY_VALUE); e == nil {
		existing = true
		k.Close()
	} else if e != registry.ErrNotExist {
		return e
	}
	if _, e := os.Stat(filepath.Join(dir, "Screenshot.exe")); e == nil {
		existing = true
	}
	if err = a.stop(); err != nil {
		return err
	}
	a.progress.Store(15)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := assets.ReadFile("Screenshot.exe")
	if err != nil {
		return err
	}
	if _, err = core.OwnedPath(dir, "Screenshot.exe"); err != nil {
		return err
	}
	if err = core.AtomicWrite(filepath.Join(dir, "Screenshot.exe"), data); err != nil {
		return err
	}
	a.progress.Store(55)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	data, err = os.ReadFile(self)
	if err != nil {
		return err
	}
	if _, err = core.OwnedPath(dir, "Uninstall.exe"); err != nil {
		return err
	}
	if err = core.AtomicWrite(filepath.Join(dir, "Uninstall.exe"), data); err != nil {
		return err
	}
	exe := filepath.Join(dir, "Screenshot.exe")
	if err = writeReg(`Software\Microsoft\Windows\CurrentVersion\Run`, map[string]string{"Screenshot": `"` + exe + `" --background`}); err != nil {
		return err
	}
	if err = writeReg(uninstallKey, map[string]string{"DisplayName": "Screenshot", "DisplayVersion": products.Version("screenshot"), "Publisher": "Sander Stripa", "InstallLocation": dir, "DisplayIcon": exe, "UninstallString": `"` + filepath.Join(dir, "Uninstall.exe") + `" --uninstall`}); err != nil {
		return err
	}
	if err = core.Shortcuts("Screenshot", dir, "Screenshot.exe", "Screenshot.exe", "--settings", o.Desktop, o.StartMenu); err != nil {
		return err
	}
	if !existing {
		k, _, e := registry.CreateKey(registry.CURRENT_USER, settingsKey, registry.SET_VALUE)
		if e != nil {
			return e
		}
		theme, light, english := uint32(1), uint32(1), uint32(0)
		if o.Theme == "dark" {
			theme = 2
			light = 0
		}
		if o.Language == "en" {
			english = 1
		}
		for n, v := range map[string]uint32{"ThemeMode": theme, "ThemeLight": light, "LanguageEN": english} {
			if e = k.SetDWordValue(n, v); e != nil {
				k.Close()
				return e
			}
		}
		k.Close()
	}
	a.dir = dir
	a.installed = true
	a.progress.Store(100)
	return nil
}
func testHotkey(vk, mod uint32) bool {
	r, _, _ := user32.NewProc("RegisterHotKey").Call(0, 998, uintptr(mod|0x4000), uintptr(vk))
	if r == 0 {
		return false
	}
	user32.NewProc("UnregisterHotKey").Call(0, 998)
	return true
}
func (a *Adapter) Action(action string, raw []byte) (any, error) {
	if !a.installed {
		return nil, fmt.Errorf("installation not complete")
	}
	switch action {
	case "hotkey-current":
		vk, mod := uint64(44), uint64(0)
		if k, e := registry.OpenKey(registry.CURRENT_USER, settingsKey, registry.QUERY_VALUE); e == nil {
			if v, _, e := k.GetIntegerValue("HotkeyVK"); e == nil {
				vk = v
			}
			if v, _, e := k.GetIntegerValue("HotkeyModifiers"); e == nil {
				mod = v
			}
			k.Close()
		}
		return map[string]uint64{"vk": vk, "mod": mod}, nil
	case "hotkey-save":
		var o struct {
			VK      uint32 `json:"vk"`
			Mod     uint32 `json:"mod"`
			Replace bool   `json:"replace"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return nil, err
		}
		if o.VK < 8 || o.VK > 254 || o.VK == 16 || o.VK == 17 || o.VK == 18 || o.Mod&^uint32(7) != 0 {
			return nil, fmt.Errorf("choose a non-modifier key")
		}
		printScreen := o.VK == 44 && o.Mod == 0
		if printScreen && !o.Replace {
			return nil, fmt.Errorf("Print Screen requires explicit confirmation")
		}
		if !printScreen && !testHotkey(o.VK, o.Mod) {
			return nil, fmt.Errorf("keyboard shortcut is already in use")
		}
		// System binding changes only in this explicitly confirmed operation.
		if printScreen {
			k, _, e := registry.CreateKey(registry.CURRENT_USER, `Control Panel\Keyboard`, registry.SET_VALUE)
			if e != nil {
				return nil, e
			}
			e = k.SetDWordValue("PrintScreenKeyForSnippingEnabled", 0)
			k.Close()
			if e != nil {
				return nil, e
			}
			var result uintptr
			user32.NewProc("SendMessageTimeoutW").Call(0xffff, 0x1a, 0, uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(`Control Panel\Keyboard`))), 2, 300, uintptr(unsafe.Pointer(&result)))
		}
		k, _, err := registry.CreateKey(registry.CURRENT_USER, settingsKey, registry.SET_VALUE)
		if err != nil {
			return nil, err
		}
		err = k.SetDWordValue("HotkeyVK", o.VK)
		if err == nil {
			err = k.SetDWordValue("HotkeyModifiers", o.Mod)
		}
		k.Close()
		if err != nil {
			return nil, err
		}
		a.hotkeyDone = true
		return map[string]bool{"relogin": !testHotkey(o.VK, o.Mod)}, nil
	}
	return nil, fmt.Errorf("unknown action")
}
func (a *Adapter) Open() error {
	if !a.installed || !a.hotkeyDone {
		return fmt.Errorf("finish hotkey configuration first")
	}
	return core.PowerShell(`(New-Object -ComObject Shell.Application).ShellExecute(` + core.QuotePS(filepath.Join(a.dir, "Screenshot.exe")) + `, '--background', '', 'open', 1)`)
}
func (a *Adapter) Remove() error {
	if _, err := core.ValidateDir(a.dir); err != nil {
		return err
	}
	if err := a.stop(); err != nil {
		return err
	}
	p, err := core.OwnedPath(a.dir, "Screenshot.exe")
	if err != nil {
		return err
	}
	if err = os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	if k, e := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE); e == nil {
		e = k.DeleteValue("Screenshot")
		k.Close()
		if e != nil && e != registry.ErrNotExist {
			return e
		}
	}
	if err = registry.DeleteKey(registry.CURRENT_USER, uninstallKey); err != nil && err != registry.ErrNotExist {
		return err
	}
	if err = core.Shortcuts("Screenshot", a.dir, "Screenshot.exe", "Screenshot.exe", "--settings", false, false); err != nil {
		return err
	}
	a.progress.Store(100)
	return nil
}
