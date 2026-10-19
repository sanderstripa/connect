//go:build windows

// Package adaptertest exercises real Windows operations only on disposable GitHub runners.
//
package adaptertest

import (
	"bytes"
	"connect-installer.local/core/core"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"testing"
)

func Lifecycle(t *testing.T, id, executable string, newAdapter func() core.Adapter) {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("CONNECT_INSTALLER_CI_TEST") != "1" {
		t.Skip("real installation test requires an explicitly enabled disposable GitHub runner")
	}
	root := t.TempDir()
	o := core.Options{Path: root, Desktop: true, StartMenu: true, Chrome: id == "connect", Language: "ru", Theme: "light"}
	a := newAdapter()
	if err := a.Install(o); err != nil {
		t.Fatalf("clean install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, executable)); err != nil {
		t.Fatal("application missing", err)
	}
	regRoot, key := registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall\`+map[string]string{"connect": "ConnectApp", "screenshot": "Screenshot", "screencast": "Screencast", "torrent": "Torrent", "soulu": "Soulu"}[id]
	if id == "connect" {
		regRoot = registry.LOCAL_MACHINE
	}
	k, err := registry.OpenKey(regRoot, key, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal("Windows registration missing", err)
	}
	location, _, err := k.GetStringValue("InstallLocation")
	k.Close()
	if err != nil || location != root {
		t.Fatal("registered installation path differs", location, err)
	}
	foreign := filepath.Join(root, "user-owned-sentinel.txt")
	sentinel := []byte("preserve user-owned bytes\n")
	if err = os.WriteFile(foreign, sentinel, 0600); err != nil {
		t.Fatal(err)
	}
	var profile string
	switch id {
	case "connect":
		profile = filepath.Join(os.Getenv("APPDATA"), "connect-app", "config.json")
	case "torrent":
		profile = filepath.Join(os.Getenv("LOCALAPPDATA"), "Torrent", "settings.txt")
	case "soulu":
		profile = filepath.Join(os.Getenv("LOCALAPPDATA"), "Soulu", "User Data", "installer-ci-sentinel.txt")
	case "screencast":
		profile = filepath.Join(root, "models", "user-owned-model-sentinel.bin")
	}
	if profile != "" {
		original, e := os.ReadFile(profile)
		if e != nil && !os.IsNotExist(e) {
			t.Fatal(e)
		}
		if err = os.MkdirAll(filepath.Dir(profile), 0755); err != nil {
			t.Fatal(err)
		}
		payload := []byte("preserve existing profile\n")
		if id == "connect" {
			payload = []byte(`{"ui":{"language":"en","theme":"dark"}}`)
		}
		if id == "torrent" {
			payload = []byte(filepath.Join(os.Getenv("USERPROFILE"), "Downloads") + "\n0 1 1\n")
		}
		if err = os.WriteFile(profile, payload, 0600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if e == nil {
				os.WriteFile(profile, original, 0600)
			} else {
				os.Remove(profile)
			}
		})
		sentinelProfile := append([]byte(nil), payload...)
		defer func() {
			got, err := os.ReadFile(profile)
			if err != nil || !bytes.Equal(got, sentinelProfile) {
				t.Error("profile/model changed during lifecycle", err)
			}
		}()
	}
	if id == "screenshot" || id == "screencast" {
		settings, value := `Software\SanderStripa\Screenshot`, "ThemeMode"
		if id == "screencast" {
			settings, value = `Software\Screencast`, "Theme"
		}
		preferences, e := registry.OpenKey(registry.CURRENT_USER, settings, registry.QUERY_VALUE|registry.SET_VALUE)
		if e != nil {
			t.Fatal(e)
		}
		previous, _, e := preferences.GetIntegerValue(value)
		if e != nil {
			preferences.Close()
			t.Fatal(e)
		}
		if e = preferences.SetDWordValue(value, 2); e != nil {
			preferences.Close()
			t.Fatal(e)
		}
		preferences.Close()
		t.Cleanup(func() {
			if k, e := registry.OpenKey(registry.CURRENT_USER, settings, registry.SET_VALUE); e == nil {
				k.SetDWordValue(value, uint32(previous))
				k.Close()
			}
		})
		defer func() {
			k, e := registry.OpenKey(registry.CURRENT_USER, settings, registry.QUERY_VALUE)
			if e != nil {
				t.Error(e)
				return
			}
			defer k.Close()
			v, _, e := k.GetIntegerValue(value)
			if e != nil || v != 2 {
				t.Error("saved appearance was reset", v, e)
			}
		}()
	}
	updater := newAdapter()
	o.Language = "en"
	o.Theme = "light"
	if err = updater.Install(o); err != nil {
		t.Fatalf("update: %v", err)
	}
	if data, e := os.ReadFile(foreign); e != nil || !bytes.Equal(data, sentinel) {
		t.Fatal("update lost a user file", e)
	}
	if err = updater.Remove(); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err = os.Stat(filepath.Join(root, executable)); !os.IsNotExist(err) {
		t.Fatal("application still present after uninstall", err)
	}
	if data, e := os.ReadFile(foreign); e != nil || !bytes.Equal(data, sentinel) {
		t.Fatal("uninstall lost a user file", e)
	}
	if k, e := registry.OpenKey(regRoot, key, registry.QUERY_VALUE); e == nil {
		k.Close()
		t.Error("Installed Apps entry remains after uninstall")
	} else if e != registry.ErrNotExist {
		t.Error(e)
	}
	t.Log("real clean install, registered-path update, preserved user data and uninstall passed")
}
