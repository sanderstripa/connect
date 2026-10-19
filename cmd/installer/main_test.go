//go:build windows && legacy_installer

package main

import (
	"bytes"
	"encoding/json"
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/installprefs"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerRejectsRequestsWithoutToken(t *testing.T) {
	s := &setup{token: "private"}
	for _, method := range []string{"GET", "POST"} {
		r := httptest.NewRequest(method, "/api/install", strings.NewReader(`{"desktop":true}`))
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
}
func TestInstallerRejectsRelativeDestination(t *testing.T) {
	if _, err := validateDir("relative/Connect"); err == nil {
		t.Fatal("must reject relative target")
	}
}
func TestAtomicUpdateReplacesOnlyNamedFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Connect.exe")
	other := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(other, []byte("keep"), 0600)
	for _, v := range []string{"old", "new"} {
		if err := writeAtomic(p, []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := os.ReadFile(p)
	if string(got) != "new" {
		t.Fatal(string(got))
	}
	got, _ = os.ReadFile(other)
	if string(got) != "keep" {
		t.Fatal("modified unrelated file")
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 2 {
		t.Fatal("temporary files left behind")
	}
}
func TestInstallerEmbedsWindowsExecutable(t *testing.T) {
	if len(payload) < 2 || string(payload[:2]) != "MZ" {
		t.Fatal("missing Windows executable")
	}
}

func isolatedMachineRegistry(t *testing.T) {
	t.Helper()
	fixture := t.TempDir()
	name := `Software\ConnectInstallerTests\` + filepath.Base(filepath.Dir(fixture)) + filepath.Base(fixture)
	key, _, err := registry.CreateKey(registry.CURRENT_USER, name, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	old := machineRegistry
	machineRegistry = key
	t.Cleanup(func() {
		machineRegistry = old
		deleteRegistryTree(t, key)
		key.Close()
		if e := registry.DeleteKey(registry.CURRENT_USER, name); e != nil {
			t.Error(e)
		}
	})
}
func deleteRegistryTree(t *testing.T, key registry.Key) {
	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		t.Error(err)
		return
	}
	for _, name := range names {
		child, e := registry.OpenKey(key, name, registry.ALL_ACCESS)
		if e != nil {
			t.Error(e)
			continue
		}
		deleteRegistryTree(t, child)
		child.Close()
		if e = registry.DeleteKey(key, name); e != nil {
			t.Error(e)
		}
	}
}
func TestWindowsInstallUpdateRemoveIsolated(t *testing.T) {
	isolatedMachineRegistry(t)
	base := t.TempDir()
	t.Setenv("ProgramFiles", filepath.Join(base, "Program Files"))
	t.Setenv("ProgramData", filepath.Join(base, "ProgramData"))
	t.Setenv("PUBLIC", filepath.Join(base, "Public"))
	if err := os.MkdirAll(filepath.Join(base, "Public", "Desktop"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, options := range []struct {
		name                  string
		desktop, menu, chrome bool
	}{{"defaults", true, true, true}, {"no-options", false, false, false}} {
		t.Run(options.name, func(t *testing.T) {
			dir := filepath.Join(base, options.name, "Custom folder's CONNECT")
			s := &setup{dir: dir, cleanupAutostart: func() error { return nil }}
			if err := s.install(options.desktop, options.chrome, options.menu, dir, installprefs.Preferences{Language: "en", Theme: "dark"}); err != nil {
				t.Fatal(err)
			}
			raw, preferenceErr := os.ReadFile(filepath.Join(dir, installprefs.File))
			var prefs installprefs.Preferences
			if preferenceErr != nil || json.Unmarshal(raw, &prefs) != nil || !prefs.Valid() || prefs.Language != "en" || prefs.Theme != "dark" {
				t.Fatal("installer preferences missing", string(raw), preferenceErr)
			}
			if installDir() != dir {
				t.Fatal("registered destination not restored")
			}
			got, _ := os.ReadFile(filepath.Join(dir, "Connect.exe"))
			if !bytes.Equal(got, payload) {
				t.Fatal("payload changed")
			}
			userFile := filepath.Join(dir, "settings.json")
			os.WriteFile(userFile, []byte("preserve"), 0600)
			extension := filepath.Join(dir, "ChromeExtension", "manifest.json")
			_, e := os.Stat(extension)
			if (e == nil) != options.chrome {
				t.Fatal("extension option ignored", e)
			}
			host, e := registry.OpenKey(machineRegistry, `Software\Google\Chrome\NativeMessagingHosts\com.sanderstripa.connect`, registry.QUERY_VALUE)
			if options.chrome {
				if e != nil {
					t.Fatal(e)
				}
				v, _, e := host.GetStringValue("")
				host.Close()
				if e != nil || v != filepath.Join(dir, "chrome-native-host.json") {
					t.Fatal("host registration", v, e)
				}
			} else if e != registry.ErrNotExist {
				t.Fatal("unexpected native host", e)
			}
			for _, p := range []struct {
				path   string
				wanted bool
			}{{filepath.Join(base, "Public", "Desktop", "Connect.lnk"), options.desktop}, {filepath.Join(base, "ProgramData", `Microsoft\Windows\Start Menu\Programs\Connect\Connect.lnk`), options.menu}} {
				_, e := os.Stat(p.path)
				if (e == nil) != p.wanted {
					t.Fatal("shortcut option ignored", p.path, e)
				}
			}
			h, e := windows.CreateFile(windows.StringToUTF16Ptr(filepath.Join(dir, "Connect.exe")), windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
			if e != nil {
				t.Fatal(e)
			}
			if e = s.install(options.desktop, options.chrome, options.menu, dir, installprefs.Preferences{Language: "en", Theme: "dark"}); e == nil {
				t.Error("locked app update accepted")
			}
			if e = s.remove(); e == nil {
				t.Error("locked app removal accepted")
			}
			windows.CloseHandle(h)
			if _, e = os.Stat(userFile); e != nil {
				t.Fatal("data lost after locked operation")
			}
			if e = s.install(options.desktop, options.chrome, options.menu, dir, installprefs.Preferences{Language: "en", Theme: "dark"}); e != nil {
				t.Fatal("update", e)
			}
			if options.chrome {
				before, e := os.Stat(extension)
				if e != nil {
					t.Fatal(e)
				}
				if e = s.install(false, false, false, dir); e != nil {
					t.Fatal(e)
				}
				after, e := os.Stat(extension)
				if e != nil || !before.ModTime().Equal(after.ModTime()) {
					t.Fatal("disabled extension was changed", e)
				}
				if s.chrome {
					t.Fatal("Chrome preparation must remain disabled")
				}
			}
			if e = s.remove(); e != nil {
				t.Fatal("remove", e)
			}
			got, _ = os.ReadFile(userFile)
			if string(got) != "preserve" {
				t.Fatal("user data lost")
			}
			if _, e = os.Stat(filepath.Join(dir, "Connect.exe")); !os.IsNotExist(e) {
				t.Fatal("exe not removed")
			}
			if s.progress.Load() != 100 {
				t.Fatal("progress incomplete")
			}
		})
	}
}
func TestAPIProgressDoesNotBlockAndChromeRequiresSelection(t *testing.T) {
	s := &setup{token: "private"}
	s.progress.Store(45)
	s.mu.Lock()
	r := httptest.NewRequest("GET", "/api/progress", nil)
	r.Header.Set("X-Setup-Token", "private")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	s.mu.Unlock()
	if w.Code != 200 || !strings.Contains(w.Body.String(), "45") {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, endpoint := range []string{"chrome", "open"} {
		r := httptest.NewRequest("POST", "/api/"+endpoint, strings.NewReader(`{}`))
		r.Header.Set("X-Setup-Token", "private")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 500 {
			t.Fatal("unexpected action allowed", endpoint)
		}
	}
}
func TestRunningExecutableIsNeverTerminated(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "Connect.exe")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go.exe"), "build", "-o", exe, "testdata/busy-app.go")
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatalf("fixture build: %v %s", e, out)
	}
	process := exec.Command(exe)
	if e := process.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { process.Process.Kill(); process.Wait() }() // Only the inert test-owned fixture.
	if e := checkAppClosed(dir); e == nil {
		t.Fatal("running executable accepted")
	}
	h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(process.Process.Pid))
	if e != nil {
		t.Fatal(e)
	}
	defer windows.CloseHandle(h)
	var code uint32
	if e = windows.GetExitCodeProcess(h, &code); e != nil || code != 259 {
		t.Fatal("installer terminated process", code, e)
	}
}
