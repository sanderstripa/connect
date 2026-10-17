// Package awgconfig parses data-only wg-quick configurations. Shell hooks are
// deliberately rejected; imported profiles never execute commands.
package awgconfig

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

type Config struct {
	Address   []string          `json:"address"`
	DNS       []string          `json:"dns"`
	MTU       int               `json:"mtu"`
	Interface map[string]string `json:"interface"`
	Peer      map[string]string `json:"peer"`
}

var deviceFields = map[string]string{
	"PrivateKey": "private_key", "Jc": "jc", "Jmin": "jmin", "Jmax": "jmax",
	"S1": "s1", "S2": "s2", "S3": "s3", "S4": "s4",
	"H1": "h1", "H2": "h2", "H3": "h3", "H4": "h4",
	"I1": "i1", "I2": "i2", "I3": "i3", "I4": "i4", "I5": "i5",
	"HeaderProtectionKey": "header_protection_key", "ContentPaddingAddition": "content_padding_addition",
	"RekeyAfterTime": "rekey_after_time", "RekeyTimeout": "rekey_timeout", "RejectAfterTime": "reject_after_time",
	"KeepaliveTimeout": "keepalive_timeout", "MaxHandshakeAttempts": "max_handshake_attempts",
	"RandomTrailers": "random_trailers", "DisableCookies": "disable_cookies",
}
var peerFields = map[string]string{"PublicKey": "public_key", "PresharedKey": "preshared_key", "Endpoint": "endpoint", "AllowedIPs": "allowed_ip", "PersistentKeepalive": "persistent_keepalive_interval"}

func IsConfig(s string) bool {
	if strings.HasPrefix(s, "awg://") || strings.HasPrefix(s, "vpn://") {
		return true
	}
	for _, line := range strings.Split(strings.TrimPrefix(s, "\ufeff"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		return strings.HasPrefix(line, "[Interface]")
	}
	return false
}
func Parse(raw string) (Config, error) {
	if len(raw) <= 1024*1024 && strings.HasPrefix(raw, "vpn://") {
		return parseShare(raw)
	}
	c := Config{MTU: 1280, Interface: map[string]string{}, Peer: map[string]string{}}
	if len(raw) > 1024*1024 {
		return c, fmt.Errorf("AmneziaWG: configuration exceeds 1 MB")
	}
	if strings.HasPrefix(raw, "awg://") {
		var b []byte
		var err error
		for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
			b, err = enc.DecodeString(strings.TrimPrefix(raw, "awg://"))
			if err == nil {
				break
			}
		}
		if err != nil {
			return c, fmt.Errorf("AmneziaWG: invalid encoded configuration")
		}
		raw = string(b)
	}
	section := ""
	for _, line := range strings.Split(strings.TrimPrefix(raw, "\ufeff"), "\n") {
		line, _, _ = strings.Cut(line, "#")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if line == "[Interface]" {
			if section != "" {
				return c, fmt.Errorf("AmneziaWG: duplicate Interface")
			}
			section = "interface"
			continue
		}
		if line == "[Peer]" {
			if section != "interface" {
				return c, fmt.Errorf("AmneziaWG: exactly one peer is required")
			}
			section = "peer"
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if !ok || v == "" || strings.ContainsAny(v, "\x00\r\n") {
			return c, fmt.Errorf("AmneziaWG: invalid configuration line")
		}
		if section == "interface" {
			switch k {
			case "Address":
				if len(c.Address) > 0 {
					return c, fmt.Errorf("AmneziaWG: duplicate Address")
				}
				c.Address = split(v)
			case "DNS":
				if len(c.DNS) > 0 {
					return c, fmt.Errorf("AmneziaWG: duplicate DNS")
				}
				c.DNS = split(v)
			case "MTU":
				n, e := strconv.Atoi(v)
				if e != nil || n < 576 || n > 9000 {
					return c, fmt.Errorf("AmneziaWG: invalid MTU")
				}
				c.MTU = n
			case "ListenPort":
				n, e := strconv.Atoi(v)
				if e != nil || n < 0 || n > 65535 {
					return c, fmt.Errorf("AmneziaWG: invalid ListenPort")
				} // Local port is allocated automatically to keep desktop and Chrome independent.
			default:
				if _, ok := deviceFields[k]; !ok {
					return c, fmt.Errorf("AmneziaWG: unsupported Interface field %s", k)
				}
				if _, ok := c.Interface[k]; ok {
					return c, fmt.Errorf("AmneziaWG: duplicate field %s", k)
				}
				c.Interface[k] = v
			}
		} else if section == "peer" {
			if _, ok := peerFields[k]; !ok {
				return c, fmt.Errorf("AmneziaWG: unsupported Peer field %s", k)
			}
			if _, ok := c.Peer[k]; ok {
				return c, fmt.Errorf("AmneziaWG: duplicate field %s", k)
			}
			c.Peer[k] = v
		} else {
			return c, fmt.Errorf("AmneziaWG: expected [Interface]")
		}
	}
	return c, c.Validate()
}
func split(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		out = append(out, strings.TrimSpace(v))
	}
	return out
}
func key(s string) (string, error) {
	b, e := base64.StdEncoding.DecodeString(s)
	if e != nil || len(b) != 32 {
		return "", fmt.Errorf("AmneziaWG: invalid key")
	}
	var nonzero byte
	for _, v := range b {
		nonzero |= v
	}
	if nonzero == 0 {
		return "", fmt.Errorf("AmneziaWG: zero key")
	}
	return hex.EncodeToString(b), nil
}
func (c Config) Validate() error {
	if len(c.Address) == 0 || len(c.Address) > 8 {
		return fmt.Errorf("AmneziaWG: Address is required")
	}
	for _, s := range c.Address {
		p, e := netip.ParsePrefix(s)
		if e != nil || p.Addr().IsUnspecified() || p.Addr().IsMulticast() {
			return fmt.Errorf("AmneziaWG: invalid Address")
		}
	}
	if c.MTU < 576 || c.MTU > 9000 {
		return fmt.Errorf("AmneziaWG: invalid MTU")
	}
	for _, s := range c.DNS {
		if _, e := netip.ParseAddr(s); e != nil {
			return fmt.Errorf("AmneziaWG: DNS must contain IP addresses")
		}
	}
	if _, e := key(c.Interface["PrivateKey"]); e != nil {
		return e
	}
	if _, e := key(c.Peer["PublicKey"]); e != nil {
		return e
	}
	if s := c.Peer["PresharedKey"]; s != "" {
		if _, e := key(s); e != nil {
			return e
		}
	}
	if s := c.Interface["HeaderProtectionKey"]; s != "" {
		if _, e := key(s); e != nil {
			return e
		}
	}
	host, port, e := net.SplitHostPort(c.Peer["Endpoint"])
	n, pe := strconv.Atoi(port)
	if e != nil || pe != nil || host == "" || n < 1 || n > 65535 {
		return fmt.Errorf("AmneziaWG: Endpoint must contain server and port")
	}
	if len(host) > 253 || strings.ContainsAny(host, " /\\\t\r\n\x00") {
		return fmt.Errorf("AmneziaWG: invalid Endpoint")
	}
	if c.Peer["AllowedIPs"] == "" {
		return fmt.Errorf("AmneziaWG: AllowedIPs is required")
	}
	for _, s := range split(c.Peer["AllowedIPs"]) {
		if _, e := netip.ParsePrefix(s); e != nil {
			return fmt.Errorf("AmneziaWG: invalid AllowedIPs")
		}
	}
	for k, v := range c.Interface {
		if _, ok := deviceFields[k]; !ok || strings.ContainsAny(v, "\r\n\x00") {
			return fmt.Errorf("AmneziaWG: invalid Interface field")
		}
	}
	for k, v := range c.Peer {
		if _, ok := peerFields[k]; !ok || strings.ContainsAny(v, "\r\n\x00") {
			return fmt.Errorf("AmneziaWG: invalid Peer field")
		}
	}
	return nil
}

// UAPI produces only allowlisted keys. The core validates obfuscation values;
// errors returned to the UI never echo the imported secrets.
func (c Config) UAPI(endpoint string) (string, error) {
	if e := c.Validate(); e != nil {
		return "", e
	}
	var b strings.Builder
	for k, target := range deviceFields {
		if v := c.Interface[k]; v != "" {
			if k == "PrivateKey" || k == "HeaderProtectionKey" {
				v, _ = key(v)
			}
			fmt.Fprintf(&b, "%s=%s\n", target, v)
		}
	}
	b.WriteString("replace_peers=true\n")
	pub, _ := key(c.Peer["PublicKey"])
	fmt.Fprintf(&b, "public_key=%s\n", pub)
	for k, target := range peerFields {
		if k == "PublicKey" {
			continue
		}
		v := c.Peer[k]
		if k == "Endpoint" {
			v = endpoint
		}
		if v == "" {
			continue
		}
		if k == "PresharedKey" {
			v, _ = key(v)
		}
		if k == "AllowedIPs" {
			for _, p := range split(v) {
				fmt.Fprintf(&b, "allowed_ip=%s\n", p)
			}
		} else {
			fmt.Fprintf(&b, "%s=%s\n", target, v)
		}
	}
	return b.String(), nil
}
