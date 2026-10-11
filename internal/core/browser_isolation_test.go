//go:build windows

package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserBackendUsesSeparateStoreAndNeverPreparesTUN(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	b, e := NewBrowserBackend(nil, "")
	if e != nil {
		t.Fatal(e)
	}
	if !b.browserOnly || filepath.Base(b.store.RootDir()) != "connect-chrome" {
		t.Fatal("browser storage isolation failed")
	}
	node, e := b.UpsertNode(NodeConfig{Name: "test", ServerAddress: "127.0.0.1:443", Key: "test", LocalPort: 1080, Enabled: true})
	if e != nil {
		t.Fatal(e)
	}
	if node.LocalPort != 17991 {
		t.Fatalf("imported local port %d", node.LocalPort)
	}
	// This request cannot activate TUN even if persisted Windows defaults enable it.
	runtimeDir := b.store.RuntimeDir()
	dir := filepath.Join(runtimeDir, "bin", "windows-amd64")
	os.MkdirAll(dir, 0700)
	for _, name := range []string{"sudoku.exe", "hev-socks5-tunnel.exe", "wintun.dll", "msys-2.0.dll"} {
		os.WriteFile(filepath.Join(dir, name), []byte("test"), 0600)
	}
	plan, e := b.prepareProxyStartLocked(StartRequest{WithTun: true})
	if e != nil {
		t.Fatal(e)
	}
	if plan.withTun {
		t.Fatal("browser helper prepared machine-wide TUN")
	}
}
