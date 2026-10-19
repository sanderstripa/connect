//go:build windows

package vpn

import (
	"connect-installer.local/core/core"
	"connect-installer.local/core/installprefs"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Adapter struct{ s *setup }

func New() *Adapter                    { return &Adapter{s: &setup{dir: installDir()}} }
func (a *Adapter) Path() string        { return a.s.dir }
func (a *Adapter) AfterRemoval() error { return core.CleanupUninstaller(a.s.dir, "Uninstall.exe") }
func existingConfiguration(destination string) bool {
	for _, p := range []string{filepath.Join(destination, "Connect.exe"), filepath.Join(destination, installprefs.File), filepath.Join(os.Getenv("APPDATA"), "connect-app", "config.json"), filepath.Join(os.Getenv("APPDATA"), "connect-chrome", "config.json"), filepath.Join(os.Getenv("APPDATA"), "connect-app-legacy", "config.json"), filepath.Join(os.Getenv("APPDATA"), "sudoku-desktop", "config.json")} {
		if _, e := os.Stat(p); e == nil {
			return true
		} else if !os.IsNotExist(e) {
			return true
		}
	}
	key, e := registry.OpenKey(machineRegistry, uninstallKey, registry.QUERY_VALUE)
	if e == nil {
		key.Close()
		return true
	}
	return e != registry.ErrNotExist
}
func (a *Adapter) Install(o core.Options) error {
	if _, e := os.Stat(filepath.Join(a.s.dir, "Connect.exe")); e == nil && !strings.EqualFold(filepath.Clean(o.Path), a.s.dir) {
		return fmt.Errorf("update the existing installation at %s", a.s.dir)
	}
	return a.s.install(o.Desktop, o.Chrome, o.StartMenu, o.Path, installprefs.Preferences{Language: o.Language, Theme: o.Theme})
}
func (a *Adapter) Remove() error { return a.s.remove() }
func (a *Adapter) Open() error {
	if !a.s.installed {
		return fmt.Errorf("installation not complete")
	}
	return exec.Command("explorer.exe", filepath.Join(a.s.dir, "Connect.exe")).Start()
}
func (a *Adapter) Action(name string, _ []byte) (any, error) {
	if name != "chrome" || !a.s.installed || !a.s.chrome {
		return nil, fmt.Errorf("invalid action")
	}
	return map[string]bool{"ok": true}, prepareChrome(a.s.dir)
}
func (a *Adapter) Progress() core.Progress { return core.Progress{Percent: int(a.s.progress.Load())} }
