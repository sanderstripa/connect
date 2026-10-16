//go:build windows

package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestManagedComponentIsUsedAndSurvivesBundledRefresh(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APPDATA", root)
	runtime := fstest.MapFS{}
	for _, name := range []string{"sudoku.exe", "xray.exe", "hev-socks5-tunnel.exe", "wintun.dll", "msys-2.0.dll"} {
		runtime["runtime/bin/windows-amd64/"+name] = &fstest.MapFile{Data: []byte("bundled " + name)}
	}
	b, e := NewBrowserBackend(runtime, "runtime/bin")
	if e != nil {
		t.Fatal(e)
	}
	node, e := ParseEngineLink("vless://00000000-0000-4000-8000-000000000001@127.0.0.1:443?type=xhttp&security=none")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.UpsertNode(*node); e != nil {
		t.Fatal(e)
	}
	slot := strings.Repeat("a", 64)
	dir := filepath.Join(root, "connect-components", "xray", slot)
	os.MkdirAll(dir, 0700)
	file := filepath.Join(dir, "xray.exe")
	os.WriteFile(file, []byte("verified managed engine"), 0600)
	sum, _ := fileSHA256(file)
	state := map[string]any{"current": map[string]any{"version": "v99", "slot": slot, "hashes": map[string]string{"xray.exe": sum}}}
	raw, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(filepath.Dir(dir), "active.json"), raw, 0600)
	for i := 0; i < 2; i++ {
		b.mu.Lock()
		plan, err := b.prepareProxyStartLocked(StartRequest{WithTun: false})
		b.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if plan.sudokuBinary != file {
			t.Fatal("managed component was replaced by bundled binary")
		}
	}
	status, e := b.GetUpdateStatus("xray", false, true)
	if e != nil || status.Installed != "v99" {
		t.Fatal("actual installed version not reported", e)
	}
	os.WriteFile(file, []byte("tampered"), 0600)
	b.mu.Lock()
	_, e = b.prepareProxyStartLocked(StartRequest{})
	b.mu.Unlock()
	if e == nil {
		t.Fatal("tampered update silently used or replaced")
	}
}

func TestManagedXraySlotCarriesRealHTTPWithoutSystemProxy(t *testing.T) {
	root, e := filepath.Abs("../../runtime/bin")
	if e != nil {
		t.Fatal(e)
	}
	xray := filepath.Join(root, "windows-amd64", "xray.exe")
	if fileMissing(xray) {
		t.Skip("built Xray required")
	}
	t.Setenv("APPDATA", t.TempDir())
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "managed Xray traffic OK") }))
	defer target.Close()
	serverPort, local := unusedSwitchPort(t), unusedSwitchPort(t)
	uuid := "00000000-0000-4000-8000-000000000001"
	dir := t.TempDir()
	file := filepath.Join(dir, "server.json")
	cfg := map[string]any{"inbounds": []any{map[string]any{"listen": "127.0.0.1", "port": serverPort, "protocol": "vless", "settings": map[string]any{"clients": []any{map[string]any{"id": uuid}}, "decryption": "none"}, "streamSettings": map[string]any{"network": "xhttp", "security": "none", "xhttpSettings": map[string]any{"path": "/managed"}}}}, "outbounds": []any{map[string]any{"protocol": "freedom"}}}
	raw, _ := json.Marshal(cfg)
	os.WriteFile(file, raw, 0600)
	server := NewManagedProcess("xray-server")
	if e = server.Start(xray, []string{"run", "-config", file}, nil, dir, nil); e != nil {
		t.Fatal(e)
	}
	defer server.Stop(time.Second)
	b, e := NewBrowserBackend(os.DirFS(root), ".")
	if e != nil {
		t.Fatal(e)
	}
	b.Startup(context.Background())
	defer b.Shutdown()
	node, e := ParseEngineLink("vless://" + uuid + "@127.0.0.1:" + fmtPort(serverPort) + "?type=xhttp&security=none&path=%2Fmanaged")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.UpsertNode(*node); e != nil {
		t.Fatal(e)
	}
	b.mu.Lock()
	b.cfg.Core.LocalPort = local
	b.cfg.Nodes[0].LocalPort = local
	b.kernelProbing = true
	b.mu.Unlock()
	m, e := b.updateManagerLocked()
	if e != nil {
		t.Fatal(e)
	}
	slot := strings.Repeat("b", 64)
	slotDir := filepath.Join(m.Root, "xray", slot)
	os.MkdirAll(slotDir, 0700)
	managed := filepath.Join(slotDir, "xray.exe")
	if e = copyFileAtomic(xray, managed, 0755); e != nil {
		t.Fatal(e)
	}
	sum, _ := fileSHA256(managed)
	raw, _ = json.Marshal(map[string]any{"current": map[string]any{"version": "managed-test", "slot": slot, "hashes": map[string]string{"xray.exe": sum}}})
	os.WriteFile(filepath.Join(filepath.Dir(slotDir), "active.json"), raw, 0600)
	if e = b.StartProxy(StartRequest{}); e != nil {
		t.Fatal(e)
	}
	if b.sysProxyRestore != nil || b.routeState != nil {
		t.Fatal("browser update touched system proxy or routes")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = waitForTCPReady(ctx, "127.0.0.1:"+fmtPort(local), 4*time.Second); e != nil {
		t.Fatal(e)
	}
	if e = checkProxyURL(ctx, target.URL, local, 4*time.Second); e != nil {
		t.Fatal(e)
	}
}
