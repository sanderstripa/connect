//go:build windows

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func unusedSwitchPort(t *testing.T) int {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

func TestSwitchKeepsTunAndRollsBackFailedCore(t *testing.T) {
	root, e := filepath.Abs("../../runtime/bin")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(root, "windows-amd64", "xray.exe")); e != nil {
		t.Skip("built engines required")
	}
	t.Setenv("APPDATA", t.TempDir())
	b, e := NewBackendWithRuntimeFS(os.DirFS(root), ".")
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "switch traffic OK") }))
	defer target.Close()
	b.preflightTarget = target.Listener.Addr().String()
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	uuid := "00000000-0000-4000-8000-000000000001"
	xp, sp, local := unusedSwitchPort(t), unusedSwitchPort(t), unusedSwitchPort(t)
	scratch := t.TempDir()
	startServer := func(name string, cfg any) *ManagedProcess {
		t.Helper()
		p := NewManagedProcess(name)
		file := filepath.Join(scratch, name+".json")
		raw, _ := json.Marshal(cfg)
		os.WriteFile(file, raw, 0600)
		args := []string{"-c", file}
		if name == "xray" {
			args = []string{"run", "-config", file}
		}
		if e := p.Start(filepath.Join(root, "windows-amd64", name+".exe"), args, nil, scratch, nil); e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { p.Stop(time.Second) })
		return p
	}
	startServer("xray", map[string]any{"inbounds": []any{map[string]any{"listen": "127.0.0.1", "port": xp, "protocol": "vless", "settings": map[string]any{"clients": []any{map[string]any{"id": uuid}}, "decryption": "none"}, "streamSettings": map[string]any{"network": "xhttp", "security": "none", "xhttpSettings": map[string]any{"path": "/switch"}}}}, "outbounds": []any{map[string]any{"protocol": "freedom"}}})
	startServer("sudoku", map[string]any{"mode": "server", "transport": "tcp", "local_port": sp, "key": key, "aead": "chacha20-poly1305", "padding_min": 5, "padding_max": 15, "ascii": "prefer_entropy", "enable_pure_downlink": true, "multiplex": "off", "httpmask": map[string]any{"disable": true, "mode": "legacy"}})
	xn, e := ParseEngineLink("vless://" + uuid + "@127.0.0.1:" + fmtPort(xp) + "?type=xhttp&security=none&path=%2Fswitch")
	if e != nil {
		t.Fatal(e)
	}
	xn.ID = "x"
	xn.LocalPort = local
	sn := NodeConfig{ID: "s", Protocol: "sudoku", ServerAddress: net.JoinHostPort("127.0.0.1", fmtPort(sp)), Key: key, PaddingMin: 5, PaddingMax: 15, EnablePureDownlink: true, LocalPort: local, Enabled: true, HTTPMask: HTTPMaskSettings{Disable: true}}
	// Let production normalization fill all protocol defaults.
	normalizeNode(&sn, local)
	b.cfg.Nodes = []NodeConfig{*xn, sn}
	b.cfg.ActiveNodeID = "x"
	b.cfg.Core.LocalPort = local
	plan, e := b.prepareProxyStartLocked(StartRequest{WithTun: true})
	if e != nil {
		t.Fatal(e)
	}
	if e = b.startSwitchedCore(ctx, plan, 1); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { b.coreProc.Stop(time.Second); b.tunProc.Stop(time.Second) })
	exe, _ := os.Executable()
	if e = b.tunProc.Start(exe, []string{"-test.run=^TestManagedProcessHelper$"}, []string{"SUDOKU_PROCESS_TEST=wait"}, "", nil); e != nil {
		t.Fatal(e)
	}
	pid := b.tunProc.PID()
	route := &routeContext{ServerIP: "127.0.0.1", DefaultGateway: "127.0.0.1", WindowsDefaultIfIndex: 1, TunIndex: 7, TunAlias: "Connect"}
	b.routeState = route
	b.runningLocalPort = local
	b.state.TunRunning = true
	for _, id := range []string{"s", "x", "s", "x"} {
		now := time.Now()
		if e = b.SwitchNode(id); e != nil {
			t.Fatal(e)
		}
		if time.Since(now) > 3*time.Second {
			t.Fatalf("switch too slow: %s", time.Since(now))
		}
		if b.tunProc.PID() != pid || b.routeState != route {
			t.Fatal("recreated TUN/routes")
		}
		client, e := newHTTPClientViaSOCKS5(net.JoinHostPort("127.0.0.1", fmtPort(local)), 3*time.Second)
		if e != nil {
			t.Fatal(e)
		}
		res, e := client.Get(target.URL)
		if e != nil {
			t.Fatal(e)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		client.CloseIdleConnections()
		if string(body) != "switch traffic OK" {
			t.Fatalf("wrong response: %s", body)
		}
		t.Logf("%s switch and HTTP traffic: %s", id, time.Since(now).Round(time.Millisecond))
	}
	broken := *xn
	broken.ID = "broken"
	broken.EngineConfig = `{"protocol":"invalid-connect-test-protocol"}`
	b.cfg.Nodes = append(b.cfg.Nodes, broken)
	if e = b.SwitchNode("broken"); e == nil {
		t.Fatal("invalid core accepted")
	}
	if b.cfg.ActiveNodeID != "x" || !b.coreProc.IsRunning() || b.tunProc.PID() != pid {
		t.Fatal("previous connection not restored")
	}
}

func fmtPort(p int) string { return fmt.Sprint(p) }
