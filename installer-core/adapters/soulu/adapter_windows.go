//go:build windows

package soulu

import (
	"archive/zip"
	"bytes"
	"connect-installer.local/core/core"
	"connect-installer.local/core/products"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
)

//go:embed payload.zip ribbon.png
var assets embed.FS

const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Soulu`

func (a *Adapter) AfterRemoval() error { return core.CleanupUninstaller(a.dir, "Uninstall Soulu.exe") }

type Adapter struct {
	dir       string
	progress  atomic.Int32
	installed bool
}

func New() *Adapter {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Soulu")
	if k, e := registry.OpenKey(registry.CURRENT_USER, `Software\Soulu`, registry.QUERY_VALUE); e == nil {
		if p, _, e := k.GetStringValue("InstallDir"); e == nil {
			dir = p
		}
		k.Close()
	}
	return &Adapter{dir: dir}
}
func (a *Adapter) Config() core.Config {
	return products.Configure("soulu", a.dir, assets)
}
func (a *Adapter) Progress() core.Progress            { return core.Progress{Percent: int(a.progress.Load())} }
func (a *Adapter) Action(string, []byte) (any, error) { return nil, fmt.Errorf("unknown action") }
func (a *Adapter) Open() error {
	if !a.installed {
		return fmt.Errorf("installation not complete")
	}
	return exec.Command("explorer.exe", filepath.Join(a.dir, "Soulu.exe")).Start()
}
func setReg(path string, values map[string]string) error {
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

func verifyCEF(raw []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return err
	}
	entries := map[string]*zip.File{}
	for _, f := range zr.File {
		entries[f.Name] = f
	}
	file := entries["cef-runtime-manifest.json"]
	if file == nil {
		return fmt.Errorf("CEF runtime manifest missing")
	}
	r, err := file.Open()
	if err != nil {
		return err
	}
	var m struct {
		Distribution string `json:"distribution"`
		Files        []struct {
			Path string `json:"path"`
			SHA  string `json:"sha256"`
		} `json:"files"`
	}
	err = json.NewDecoder(r).Decode(&m)
	r.Close()
	if err != nil {
		return err
	}
	if m.Distribution != "154.0.33+ga03e714+chromium-154.0.8037.94" || len(m.Files) < 5 {
		return fmt.Errorf("unexpected CEF distribution")
	}
	for _, item := range m.Files {
		f := entries[item.Path]
		if f == nil {
			return fmt.Errorf("CEF file missing: %s", item.Path)
		}
		r, e := f.Open()
		if e != nil {
			return e
		}
		hash := sha256.New()
		_, e = io.Copy(hash, r)
		r.Close()
		if e != nil {
			return e
		}
		if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), item.SHA) {
			return fmt.Errorf("CEF file hash mismatch: %s", item.Path)
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
	profile := strings.ToLower(filepath.Join(os.Getenv("LOCALAPPDATA"), "Soulu"))
	d := strings.ToLower(dir)
	if d == profile || strings.HasPrefix(d, profile+string(os.PathSeparator)) {
		return fmt.Errorf("the browser profile cannot be used as an installation folder")
	}
	if _, e := os.Stat(filepath.Join(a.dir, "Soulu.exe")); e == nil && !strings.EqualFold(dir, a.dir) {
		return fmt.Errorf("update the existing installation at %s", a.dir)
	}
	if entries, e := os.ReadDir(dir); e == nil && len(entries) > 0 {
		if _, e := os.Stat(filepath.Join(dir, "Soulu.exe")); e != nil {
			return fmt.Errorf("choose an empty folder or existing Soulu installation")
		}
	}
	// Refuse a running browser. Never invoke taskkill; the user closes it normally.
	if err = core.CheckClosed(filepath.Join(dir, "Soulu.exe")); err != nil {
		return err
	}
	raw, err := assets.ReadFile("payload.zip")
	if err != nil {
		return err
	}
	if err = verifyCEF(raw); err != nil {
		return err
	}
	a.progress.Store(15)
	if err = core.Unpack(raw, dir, "soulu", "Soulu.exe", func(p int) { a.progress.Store(int32(p)) }); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	if _, err = core.OwnedPath(dir, "Uninstall Soulu.exe"); err != nil {
		return err
	}
	if err = core.AtomicWrite(filepath.Join(dir, "Uninstall Soulu.exe"), data); err != nil {
		return err
	}
	if err = setReg(`Software\Soulu`, map[string]string{"InstallDir": dir}); err != nil {
		return err
	}
	exe := filepath.Join(dir, "Soulu.exe")
	icon := filepath.Join(dir, "ui", "soulu-icon-v20261018.ico")
	if _, e := os.Stat(icon); e != nil {
		icon = filepath.Join(dir, "ui", "soulu-icon.ico")
	}
	if err = core.Shortcuts("Soulu", dir, "Soulu.exe", strings.TrimPrefix(icon, dir+string(os.PathSeparator)), "", o.Desktop, o.StartMenu); err != nil {
		return err
	}
	if err = setReg(uninstallKey, map[string]string{"DisplayName": "Soulu", "DisplayVersion": products.Version("soulu"), "Publisher": "Soulu", "InstallLocation": dir, "DisplayIcon": exe, "UninstallString": `"` + filepath.Join(dir, "Uninstall Soulu.exe") + `" --uninstall`}); err != nil {
		return err
	}
	// Mirror RegisterDefaultBrowser in onboarding.cc, without selecting a default.
	for _, item := range []struct {
		key    string
		values map[string]string
	}{
		{`Software\Classes\Soulu.Url`, map[string]string{"": "Soulu Web Link", "URL Protocol": ""}},
		{`Software\Classes\Soulu.Url\DefaultIcon`, map[string]string{"": exe + ",0"}},
		{`Software\Classes\Soulu.Url\shell\open\command`, map[string]string{"": `"` + exe + `" "%1"`}},
		{`Software\Soulu\Capabilities`, map[string]string{"ApplicationName": "Soulu", "ApplicationDescription": "Soulu browser", "ApplicationIcon": exe + ",0"}},
		{`Software\Soulu\Capabilities\URLAssociations`, map[string]string{"http": "Soulu.Url", "https": "Soulu.Url"}},
		{`Software\RegisteredApplications`, map[string]string{"Soulu": `Software\Soulu\Capabilities`}},
	} {
		if err = setReg(item.key, item.values); err != nil {
			return err
		}
	}
	// All profile, theme, language, history and extension data remain untouched.
	a.dir = dir
	a.installed = true
	a.progress.Store(100)
	return nil
}
func (a *Adapter) Remove() error {
	if err := core.RemoveOwned(a.dir, "soulu", "Soulu.exe"); err != nil {
		return err
	}
	if err := core.Shortcuts("Soulu", a.dir, "Soulu.exe", "ui/soulu-icon.ico", "", false, false); err != nil {
		return err
	}
	script := `foreach($p in @('HKCU:\Software\Classes\Soulu.Url','HKCU:\Software\Soulu\Capabilities','HKCU:\` + uninstallKey + `')){if(Test-Path -LiteralPath $p){Remove-Item -LiteralPath $p -Recurse}};Remove-ItemProperty -LiteralPath 'HKCU:\Software\RegisteredApplications' -Name 'Soulu' -ErrorAction SilentlyContinue;Remove-ItemProperty -LiteralPath 'HKCU:\Software\Soulu' -Name 'InstallDir' -ErrorAction SilentlyContinue;`
	if err := core.PowerShell(script); err != nil {
		return err
	}
	a.progress.Store(100)
	return nil
}
