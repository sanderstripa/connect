package awgconfig

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
)

// Amnezia exports vpn:// as URL-safe Base64 of Qt qCompress JSON (four-byte
// uncompressed length followed by zlib). Only complete, local AWG profiles are
// accepted; no subscription credentials are sent to external APIs.
func parseShare(raw string) (Config, error) {
	var empty Config
	fail := func() (Config, error) {
		return empty, fmt.Errorf("AmneziaWG: this key needs an Amnezia service or has no complete AWG profile; export an AmneziaWG .conf file")
	}
	var data []byte
	var err error
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		data, err = enc.DecodeString(strings.TrimPrefix(raw, "vpn://"))
		if err == nil {
			break
		}
	}
	if err != nil || len(data) < 5 {
		return fail()
	}
	if data[0] != '{' {
		size := binary.BigEndian.Uint32(data[:4])
		if size == 0 || size > 1024*1024 {
			return fail()
		}
		r, e := zlib.NewReader(bytes.NewReader(data[4:]))
		if e != nil {
			return fail()
		}
		data, e = io.ReadAll(io.LimitReader(r, 1024*1024+1))
		r.Close()
		if e != nil || len(data) != int(size) {
			return fail()
		}
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(data, &doc) != nil {
		return fail()
	}
	var containers []map[string]json.RawMessage
	if json.Unmarshal(doc["containers"], &containers) != nil {
		return fail()
	}
	var found *Config
	str := func(m map[string]json.RawMessage, k string) string {
		var s string
		if json.Unmarshal(m[k], &s) == nil {
			return s
		}
		var n json.Number
		if json.Unmarshal(m[k], &n) == nil {
			return n.String()
		}
		return ""
	}
	for _, container := range containers {
		var awg map[string]json.RawMessage
		if json.Unmarshal(container["awg"], &awg) != nil {
			continue
		}
		last := str(awg, "last_config")
		var values map[string]json.RawMessage
		if strings.HasPrefix(strings.TrimSpace(last), "[Interface]") {
			cfg, e := Parse(last)
			if e != nil {
				return fail()
			}
			if found != nil {
				return fail()
			}
			found = &cfg
			continue
		}
		if json.Unmarshal([]byte(last), &values) != nil {
			continue
		}
		if native := str(values, "config"); strings.HasPrefix(strings.TrimSpace(native), "[Interface]") {
			cfg, e := Parse(native)
			if e != nil {
				return fail()
			}
			if found != nil {
				return fail()
			}
			found = &cfg
			continue
		}
		host := str(values, "hostName")
		if host == "" {
			host = str(doc, "hostName")
		}
		port := str(values, "port")
		if port == "" {
			port = str(awg, "port")
		}
		address := str(values, "client_ip")
		if p, e := netip.ParseAddr(address); e == nil {
			if p.Is4() {
				address += "/32"
			} else {
				address += "/128"
			}
		}
		var b strings.Builder
		b.WriteString("[Interface]\nAddress = " + address + "\nPrivateKey = " + str(values, "client_priv_key") + "\n")
		for k := range deviceFields {
			if k == "PrivateKey" {
				continue
			}
			v := str(values, k)
			if v == "" {
				v = str(awg, k)
			}
			if v != "" {
				b.WriteString(k + " = " + v + "\n")
			}
		}
		if v := str(values, "mtu"); v != "" {
			b.WriteString("MTU = " + v + "\n")
		}
		dns := []string{}
		for _, k := range []string{"dns1", "dns2"} {
			v := str(doc, k)
			if v != "" {
				dns = append(dns, v)
			}
		}
		if len(dns) > 0 {
			b.WriteString("DNS = " + strings.Join(dns, ",") + "\n")
		}
		b.WriteString("[Peer]\nPublicKey = " + str(values, "server_pub_key") + "\nEndpoint = " + net.JoinHostPort(host, port) + "\n")
		if v := str(values, "psk_key"); v != "" {
			b.WriteString("PresharedKey = " + v + "\n")
		}
		if v := str(values, "persistent_keep_alive"); v != "" {
			b.WriteString("PersistentKeepalive = " + v + "\n")
		}
		var allowed []string
		if json.Unmarshal(values["allowed_ips"], &allowed) != nil || len(allowed) == 0 {
			return fail()
		}
		b.WriteString("AllowedIPs = " + strings.Join(allowed, ",") + "\n")
		cfg, e := Parse(b.String())
		if e != nil {
			return fail()
		}
		if found != nil {
			return fail()
		}
		found = &cfg
	}
	if found == nil {
		return fail()
	}
	return *found, nil
}
