package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/awgconfig"
)

func TestAWGImportAndRuntimePreserveConfiguration(t *testing.T) {
	key := func() string { b := make([]byte, 32); rand.Read(b); return base64.StdEncoding.EncodeToString(b) }
	raw := "[Interface]\nPrivateKey = " + key() + "\nAddress = 10.8.0.2/32\nDNS = 10.8.0.1\nJc = 4\nS1 = 32\nS2 = 48\nH1 = 12345\nH2 = 23456\nH3 = 34567\nH4 = 45678\n[Peer]\nPublicKey = " + key() + "\nEndpoint = 127.0.0.1:51820\nAllowedIPs = 0.0.0.0/0\n"
	n, e := ParseImportLink(raw)
	if e != nil {
		t.Fatal(e)
	}
	if n.Protocol != "amneziawg" || n.ServerAddress != "127.0.0.1:51820" || engineName(*n) != "awg-client" {
		t.Fatal("wrong engine selection")
	}
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, e := NewStore("awg-test")
	if e != nil {
		t.Fatal(e)
	}
	cfg := DefaultConfig(store.RuntimeDir())
	file, _, port, e := writeRuntimeConfigs(store, cfg, *n, "")
	if e != nil {
		t.Fatal(e)
	}
	if port != cfg.Core.LocalPort {
		t.Fatal("wrong SOCKS port")
	}
	data, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	var parsed awgconfig.Config
	if json.Unmarshal(data, &parsed) != nil || parsed.Validate() != nil || parsed.Interface["S1"] != "32" || parsed.Peer["Endpoint"] != "127.0.0.1:51820" {
		t.Fatal("runtime lost AWG parameters")
	}
	args := engineArgs(*n, file, port)
	if len(args) != 4 || args[0] != "-config" || args[2] != "-socks" {
		t.Fatal("wrong engine arguments")
	}
}
