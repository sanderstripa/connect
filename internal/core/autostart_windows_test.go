//go:build windows

package core

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
	"os"
	"testing"
	"time"
)

func TestAutostartQuotesExecutable(t *testing.T) {
	exe := `C:\Users\O'Neil\My VPN\Sudoku.exe`
	if got := windowsAutostartCommand(exe); got != `"`+exe+`" --autostart` {
		t.Fatal(got)
	}
}

func TestAutostartRegistryRoundTrip(t *testing.T) {
	path := fmt.Sprintf(`Software\Sudoku-Test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	defer registry.DeleteKey(registry.CURRENT_USER, path)
	exe := `C:\Program Files\Sudoku\Sudoku.exe`
	if err := setWindowsRunEntry(path, "test", exe, true); err != nil {
		t.Fatal(err)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	actual, _, err := key.GetStringValue("test")
	key.Close()
	if err != nil || actual != windowsAutostartCommand(exe) {
		t.Fatalf("readback: %q %v", actual, err)
	}
	if err := setWindowsRunEntry(path, "test", exe, false); err != nil {
		t.Fatal(err)
	}
	if err := setWindowsRunEntry(path, "test", exe, false); err != nil {
		t.Fatal(err)
	}
}
