//go:build windows

package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
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
	t.Setenv("ProgramFiles", "relative")
	if _, err := safeDir(); err == nil {
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
