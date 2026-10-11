package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestXhttpImportPreservesSettings(t *testing.T) {
	n, e := ParseEngineLink("vless://00000000-0000-0000-0000-000000000001@example.com:443?type=xhttp&security=tls&sni=cover.example&host=cdn.example&path=%2Fassets&mode=packet-up&extra=%7B%22scMaxEachPostBytes%22%3A1000000%7D#My%20Xray")
	if e != nil {
		t.Fatal(e)
	}
	var cfg map[string]any
	json.Unmarshal([]byte(n.EngineConfig), &cfg)
	stream := cfg["streamSettings"].(map[string]any)
	x := stream["xhttpSettings"].(map[string]any)
	if n.Protocol != "xray" || x["path"] != "/assets" || x["mode"] != "packet-up" || x["host"] != "cdn.example" || stream["tlsSettings"].(map[string]any)["allowInsecure"] != false {
		t.Fatal(n.EngineConfig)
	}
	if !strings.Contains(n.EngineConfig, "scMaxEachPostBytes") {
		t.Fatal("lost extra")
	}
}
func TestEngineImportRejectsBadInputs(t *testing.T) {
	for _, s := range []string{"vless://key@example.com:0", "vless://key@example.com:443?security=reality", "vless://key@example.com:443?type=unknown", `{"token":"secret","endpoint":{"dial_address":"host"}}`} {
		if _, e := ParseEngineLink(s); e == nil {
			t.Errorf("accepted %s", s)
		}
	}
}
func TestMistImport(t *testing.T) {
	n, e := ParseEngineLink(`{"token":"test","endpoints":[{"authority":"example.com","dial_addresses":["127.0.0.1:443"],"path":"/mist"}]}`)
	if e != nil || n.Protocol != "mist" || n.ServerAddress != "127.0.0.1:443" {
		t.Fatalf("%+v %v", n, e)
	}
}
