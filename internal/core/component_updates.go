package core

import (
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/updates"
)

type componentStamp struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Build   string `json:"build"`
}

func (b *Backend) updateManagerLocked() (*updates.Manager, error) {
	root, e := os.UserConfigDir()
	if e != nil {
		return nil, e
	}
	dir := filepath.Dir(defaultBinaryPath(b.store.RuntimeDir(), "xray"))
	base := map[string]updates.Baseline{"app": {Version: updates.AppVersion}, "geo": {Files: map[string]string{"geoip.dat": filepath.Join(dir, "geoip.dat"), "geosite.dat": filepath.Join(dir, "geosite.dat")}}}
	if exe, e := os.Executable(); e == nil {
		base["app"] = updates.Baseline{Version: updates.AppVersion, Executable: exe, Files: map[string]string{"Connect-Setup.exe": filepath.Join(filepath.Dir(exe), "Uninstall.exe")}}
	}
	for id, name := range map[string]string{"sudoku": "sudoku.exe", "xray": "xray.exe", "mist": "mist-client.exe"} {
		file := filepath.Join(dir, name)
		if id == "sudoku" {
			file = b.cfg.Core.SudokuBinary
		}
		stamps := map[string]componentStamp{}
		if data, e := os.ReadFile(filepath.Join(filepath.Dir(file), "components.json")); e == nil {
			_ = json.Unmarshal(data, &stamps)
		}
		version := ""
		if stamp, ok := stamps[name]; ok {
			if sum, e := fileSHA256(file); e == nil && strings.EqualFold(sum, stamp.SHA256) {
				version = stamp.Version
			}
		}
		if version == "" {
			if info, e := buildinfo.ReadFile(file); e == nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			} else if sum, e := fileSHA256(file); e == nil {
				version = "SHA256 " + sum[:12]
			}
		}
		base[id] = updates.Baseline{Version: version, Build: stamps[name].Build, Files: map[string]string{name: file}}
	}
	if ip, e := fileSHA256(base["geo"].Files["geoip.dat"]); e == nil {
		if site, e := fileSHA256(base["geo"].Files["geosite.dat"]); e == nil {
			geo := base["geo"]
			geo.Version = "SHA256 " + ip[:8] + " / " + site[:8]
			base["geo"] = geo
		}
	}
	return updates.New(filepath.Join(root, "connect-components"), base), nil
}
func (b *Backend) GetUpdateStatus(id string, check, preview bool) (updates.Component, error) {
	b.mu.Lock()
	if e := b.ensureCoreBinariesLocked(); e != nil {
		b.mu.Unlock()
		return updates.Component{}, e
	}
	m, e := b.updateManagerLocked()
	b.mu.Unlock()
	if e != nil {
		return updates.Component{}, e
	}
	return m.Status(id, check, preview), nil
}
func (b *Backend) UpdateComponent(id, version string) (updates.Result, error) {
	b.mu.Lock()
	m, e := b.updateManagerLocked()
	running := b.state.Running
	b.mu.Unlock()
	if e != nil {
		return updates.Result{}, e
	}
	if id == "app" && running {
		return updates.Result{}, fmt.Errorf("Disconnect VPN before updating the application")
	}
	return m.Install(id, version)
}
func (b *Backend) RollbackComponent(id string) (updates.Result, error) {
	b.mu.Lock()
	m, e := b.updateManagerLocked()
	running := b.state.Running
	b.mu.Unlock()
	if e != nil {
		return updates.Result{}, e
	}
	if id == "app" && running {
		return updates.Result{}, fmt.Errorf("Disconnect VPN before updating the application")
	}
	return m.Rollback(id)
}
func (b *Backend) resolveComponentLocked(id, name, fallback string) (string, error) {
	m, e := b.updateManagerLocked()
	if e != nil {
		return "", e
	}
	return m.Resolve(id, name, fallback)
}
func (b *Backend) componentEnvironment(protocol string) ([]string, error) {
	if protocol != "xray" {
		return nil, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	dir := filepath.Dir(defaultBinaryPath(b.store.RuntimeDir(), "xray"))
	file, e := b.resolveComponentLocked("geo", "geoip.dat", filepath.Join(dir, "geoip.dat"))
	if e != nil {
		return nil, e
	}
	if fileMissing(file) {
		return nil, nil
	}
	return []string{"XRAY_LOCATION_ASSET=" + filepath.Dir(file)}, nil
}
