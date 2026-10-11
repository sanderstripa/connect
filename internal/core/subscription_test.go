package core

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const panelSubscription = `mixed-port: 7890
proxies:
  - name: "Мой сервер"
    type: sudoku
    server: "2001:db8::1"
    port: 443
    key: "test-private-key"
    aead-method: chacha20-poly1305
    padding-min: 7
    padding-max: 21
    table-type: prefer_ascii
    multiplex: auto
    httpmask:
      disable: false
      mode: stream
      tls: true
      host: example.com
      path-root: /vpn
    enable-pure-downlink: true
`

func TestPanelSubscriptionImportAndRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(panelSubscription)) }))
	defer server.Close()
	root := t.TempDir()
	store := &Store{rootDir: root, configPath: filepath.Join(root, "config.json"), runtimeDir: filepath.Join(root, "runtime"), logDir: filepath.Join(root, "logs")}
	b := &Backend{store: store, cfg: DefaultConfig(store.runtimeDir)}
	n, err := b.ImportShortLink("", server.URL+"/sub/token.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if n.ServerAddress != "[2001:db8::1]:443" || n.Name != "Мой сервер" || n.PaddingMin != 7 || n.PaddingMax != 21 || n.ASCII != "prefer_ascii" || !n.EnablePureDownlink || !n.HTTPMask.TLS || n.HTTPMask.PathRoot != "/vpn" || n.Multiplex != "auto" {
		t.Fatalf("panel fields lost: %+v", n)
	}
	updated, err := b.ImportShortLink("Обновлён", server.URL+"/sub/token.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != n.ID || len(b.cfg.Nodes) != 1 || b.cfg.ActiveNodeID != n.ID {
		t.Fatal("refresh should preserve selection and avoid duplicates")
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Nodes[0].SourceURL != server.URL+"/sub/token.yaml" {
		t.Fatal("subscription URL not persisted")
	}
}

func TestSubscriptionRejectsUnsupportedInput(t *testing.T) {
	for _, raw := range []string{"<html>Login</html>", "proxies: []", strings.Replace(panelSubscription, "type: sudoku", "type: vmess", 1), strings.Replace(panelSubscription, "port: 443", "port: 65536", 1)} {
		if _, err := parseSubscription([]byte(raw)); err == nil {
			t.Fatal("invalid subscription accepted")
		}
	}
	for _, link := range []string{"file:///etc/passwd", "https://user:password@example.com/sub/x.yaml", "javascript:alert(1)", "text"} {
		if _, err := ParseImportLink(link); err == nil {
			t.Fatal("invalid URL accepted")
		}
	}
}

func TestSubscriptionHTTPFailuresAndLimits(t *testing.T) {
	for _, status := range []int{http.StatusGone, http.StatusNotFound, http.StatusUnauthorized} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		_, err := ParseImportLink(server.URL)
		server.Close()
		if err == nil {
			t.Fatal("HTTP failure accepted")
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(strings.Repeat("a", 1024*1024+1))) }))
	defer server.Close()
	if _, err := ParseImportLink(server.URL); err == nil {
		t.Fatal("oversized subscription accepted")
	}
}
