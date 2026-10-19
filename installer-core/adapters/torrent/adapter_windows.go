//go:build windows

package torrent

import (
	"connect-installer.local/core/core"
	"connect-installer.local/core/products"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed payload.exe ribbon.png
var assets embed.FS

var payloadSHA string

const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Torrent`

type Adapter struct {
	dir       string
	mu        sync.Mutex
	progress  core.Progress
	installed bool
}

func New() *Adapter {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Torrent")
	if k, e := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE); e == nil {
		if p, _, e := k.GetStringValue("InstallLocation"); e == nil && p != "" {
			dir = p
		} else if p, _, e := k.GetStringValue("UninstallString"); e == nil {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, `"`) {
				p = strings.SplitN(p[1:], `"`, 2)[0]
			}
			if filepath.IsAbs(p) {
				dir = filepath.Dir(p)
			}
		}
		k.Close()
	}
	return &Adapter{dir: dir}
}
func (a *Adapter) Config() core.Config {
	return products.Configure("torrent", a.dir, assets)
}
func (a *Adapter) Progress() core.Progress { a.mu.Lock(); defer a.mu.Unlock(); return a.progress }
func (a *Adapter) set(p int, indeterminate bool) {
	a.mu.Lock()
	a.progress = core.Progress{Percent: p, Indeterminate: indeterminate}
	a.mu.Unlock()
}
func (a *Adapter) Action(string, []byte) (any, error) { return nil, fmt.Errorf("unknown action") }
func (a *Adapter) Open() error {
	if !a.installed {
		return fmt.Errorf("installation not complete")
	}
	return exec.Command("explorer.exe", filepath.Join(a.dir, "Torrent.exe")).Start()
}
func checkTree(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			_, err = core.ValidateDir(path)
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		_, err = core.OwnedPath(dir, rel)
		return err
	})
}
func (a *Adapter) Install(o core.Options) error {
	a.set(0, true)
	dir, err := core.ValidateDir(o.Path)
	if err != nil {
		return err
	}
	if _, e := os.Stat(filepath.Join(a.dir, "Torrent.exe")); e == nil && !strings.EqualFold(dir, a.dir) {
		return fmt.Errorf("update the existing installation at %s", a.dir)
	}
	if entries, e := os.ReadDir(dir); e == nil && len(entries) > 0 {
		if _, e := os.Stat(filepath.Join(dir, "Torrent.exe")); e != nil {
			return fmt.Errorf("choose an empty folder or existing Torrent installation")
		}
	}
	if err = checkTree(dir); err != nil {
		return err
	}
	if err = core.CheckClosed(filepath.Join(dir, "Torrent.exe")); err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "connect-torrent-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	data, err := assets.ReadFile("payload.exe")
	if err != nil {
		return err
	}
	payload := filepath.Join(temp, "payload.exe")
	if err = os.WriteFile(payload, data, 0700); err != nil {
		return err
	}
	progress := filepath.Join(temp, "progress.txt")
	desktop, menu := "0", "0"
	if o.Desktop {
		desktop = "1"
	}
	if o.StartMenu {
		menu = "1"
	}
	// NSIS requires /D to be last and unquoted, including paths with spaces.
	cmd := exec.Command(payload)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: `"` + payload + `" /S /PROGRESSFILE="` + progress + `" /DESKTOPSHORTCUT=` + desktop + ` /STARTMENUSHORTCUT=` + menu + ` /D=` + dir}
	if err = cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTicker(150 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case err = <-done:
			goto complete
		case <-timer.C:
			if raw, e := os.ReadFile(progress); e == nil {
				if p, e := strconv.Atoi(strings.TrimSpace(string(raw))); e == nil && p >= 0 {
					if p > 99 {
						p = 99
					}
					a.set(p, false)
				}
			}
		}
	}
complete:
	if err != nil {
		return err
	}
	if _, err = os.Stat(filepath.Join(dir, "Torrent.exe")); err != nil {
		return fmt.Errorf("NSIS completed without the application: %w", err)
	}
	installed, err := os.ReadFile(filepath.Join(dir, "Torrent.exe"))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(installed)
	if payloadSHA == "" || hex.EncodeToString(digest[:]) != payloadSHA {
		return fmt.Errorf("installed Torrent executable does not match the selected release payload")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	err = key.SetStringValue("InstallLocation", dir)
	key.Close()
	if err != nil {
		return err
	}
	a.dir = dir
	a.installed = true
	a.set(100, false)
	return nil
}
func (a *Adapter) Remove() error {
	dir, err := core.ValidateDir(a.dir)
	if err != nil {
		return err
	}
	if err = checkTree(dir); err != nil {
		return err
	}
	if err = core.CheckClosed(filepath.Join(dir, "Torrent.exe")); err != nil {
		return err
	}
	// Use the existing NSIS uninstaller, retaining associations and removal semantics.
	cmd := exec.Command(filepath.Join(dir, "Uninstall.exe"), "/S")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	a.set(0, true)
	if err = cmd.Run(); err != nil {
		return err
	}
	for i := 0; i < 100; i++ {
		_, fileErr := os.Stat(filepath.Join(dir, "Torrent.exe"))
		key, regErr := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE)
		if regErr == nil {
			key.Close()
		}
		if os.IsNotExist(fileErr) && regErr == registry.ErrNotExist {
			a.set(100, false)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("NSIS removal has not completed; check the application and Windows registration")
}
