package awgconfig

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

func testProfile() string {
	key := func() string { b := make([]byte, 32); rand.Read(b); return base64.StdEncoding.EncodeToString(b) }
	return "[Interface]\nPrivateKey = " + key() + "\nAddress = 10.8.0.2/32, fd00::2/128\nDNS = 10.8.0.1\nMTU = 1280\nJc = 4\nJmin = 40\nJmax = 80\nS1 = 32\nS2 = 48\nS3 = 16\nS4 = 24\nH1 = 12345-12350\nH2 = 23456\nH3 = 34567\nH4 = 45678\nI1 = <b 0x1234><r 16>\n[Peer]\nPublicKey = " + key() + "\nPresharedKey = " + key() + "\nEndpoint = server.example:51820\nAllowedIPs = 0.0.0.0/0, ::/0\nPersistentKeepalive = 25-35\n"
}
func TestConfigAndShareImport(t *testing.T) {
	raw := testProfile()
	c, e := Parse(raw)
	if e != nil {
		t.Fatal(e)
	}
	u, e := c.UAPI("127.0.0.1:51820")
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range []string{"s4=24", "h1=12345-12350", "i1=<b 0x1234><r 16>", "persistent_keepalive_interval=25-35", "allowed_ip=::/0", "endpoint=127.0.0.1:51820"} {
		if !strings.Contains(u, v) {
			t.Fatal("parameter lost", v)
		}
	}
	encoded := "awg://" + base64.RawURLEncoding.EncodeToString([]byte(raw))
	if _, e = Parse(encoded); e != nil {
		t.Fatal(e)
	}
	doc := map[string]any{"containers": []any{map[string]any{"awg": map[string]any{"last_config": raw}}}}
	data, _ := json.Marshal(doc)
	var packed bytes.Buffer
	binary.Write(&packed, binary.BigEndian, uint32(len(data)))
	z := zlib.NewWriter(&packed)
	z.Write(data)
	z.Close()
	share := "vpn://" + base64.RawURLEncoding.EncodeToString(packed.Bytes())
	if !IsConfig(share) {
		t.Fatal("share not detected")
	}
	if _, e = Parse(share); e != nil {
		t.Fatal(e)
	}
	last := map[string]any{"client_priv_key": c.Interface["PrivateKey"], "server_pub_key": c.Peer["PublicKey"], "psk_key": c.Peer["PresharedKey"], "client_ip": "10.8.0.2", "allowed_ips": []string{"0.0.0.0/0"}, "Jc": "4", "S1": "32", "S2": "48", "H1": "12345", "H2": "23456", "H3": "34567", "H4": "45678"}
	j, _ := json.Marshal(last)
	doc = map[string]any{"hostName": "server.example", "dns1": "10.8.0.1", "containers": []any{map[string]any{"awg": map[string]any{"port": "51820", "last_config": string(j)}}}}
	data, _ = json.Marshal(doc)
	if _, e = Parse("vpn://" + base64.RawURLEncoding.EncodeToString(data)); e != nil {
		t.Fatal(e)
	}
}
func TestRejectUnsafeOrIncompleteProfiles(t *testing.T) {
	raw := testProfile()
	for _, s := range []string{strings.Replace(raw, "MTU = 1280", "PreUp = arbitrary-command", 1), raw + "[Peer]\nPublicKey=x\n", strings.Replace(raw, "Address =", "Unknown =", 1), strings.Replace(raw, "server.example:51820", "server.example:0", 1), strings.Replace(raw, "PrivateKey = ", "PrivateKey = invalid", 1), strings.Replace(raw, "AllowedIPs = 0.0.0.0/0, ::/0", "AllowedIPs = invalid", 1), raw + "PostDown = command\n", "vpn://invalid", strings.Repeat("a", 1024*1024+1)} {
		if _, e := Parse(s); e == nil {
			t.Fatal("accepted unsafe/incomplete profile")
		}
	}
	// JSON imported into a runtime file cannot bypass the UAPI field allowlist.
	c, e := Parse(raw)
	if e != nil {
		t.Fatal(e)
	}
	c.Interface["PrivateKey"] += "\npublic_key=attacker"
	if _, e = c.UAPI("127.0.0.1:1"); e == nil {
		t.Fatal("accepted UAPI injection")
	}
}
