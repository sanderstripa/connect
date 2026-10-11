package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func protocolOf(n NodeConfig) string {
	if n.Protocol == "" {
		return "sudoku"
	}
	return n.Protocol
}
func engineName(n NodeConfig) string {
	if protocolOf(n) == "mist" {
		return "mist-client"
	}
	return "xray"
}
func engineArgs(n NodeConfig, config string, port int) []string {
	switch protocolOf(n) {
	case "xray":
		return []string{"run", "-config", config}
	case "mist":
		return []string{"-config", config, "-socks", fmt.Sprintf("127.0.0.1:%d", port)}
	}
	return []string{"-c", config}
}
func decodeBase64(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		if b, e := enc.DecodeString(s); e == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("некорректный Base64")
}

// Import transport settings without dropping xhttp mode, path, host or extra.
func ParseEngineLink(link string) (*NodeConfig, error) {
	if len(link) > 1024*1024 {
		return nil, fmt.Errorf("конфигурация слишком большая: ограничение 1 МБ")
	}
	if strings.HasPrefix(link, "mist://") {
		b, e := decodeBase64(strings.TrimPrefix(link, "mist://"))
		if e != nil {
			return nil, e
		}
		link = string(b)
	}
	if strings.HasPrefix(link, "{") {
		var cfg map[string]any
		if e := json.Unmarshal([]byte(link), &cfg); e != nil {
			return nil, fmt.Errorf("некорректный JSON Mist")
		}
		token, _ := cfg["token"].(string)
		if token == "" {
			return nil, fmt.Errorf("Mist: нужен token")
		}
		var endpoint map[string]any
		if eps, ok := cfg["endpoints"].([]any); ok && len(eps) > 0 {
			endpoint, _ = eps[0].(map[string]any)
		} else {
			endpoint, _ = cfg["endpoint"].(map[string]any)
		}
		address, _ := endpoint["dial_address"].(string)
		if address == "" {
			if addresses, ok := endpoint["dial_addresses"].([]any); ok && len(addresses) > 0 {
				address, _ = addresses[0].(string)
			}
		}
		if _, _, e := net.SplitHostPort(address); e != nil {
			return nil, fmt.Errorf("Mist: нужен dial_address с портом")
		}
		return &NodeConfig{Protocol: "mist", Name: "Mist", ServerAddress: address, Key: token, EngineConfig: link, Enabled: true}, nil
	}
	if strings.HasPrefix(link, "vmess://") {
		b, e := decodeBase64(strings.TrimPrefix(link, "vmess://"))
		if e != nil {
			return nil, e
		}
		var v map[string]any
		if e = json.Unmarshal(b, &v); e != nil {
			return nil, e
		}
		str := func(k string) string {
			if v[k] == nil {
				return ""
			}
			return fmt.Sprint(v[k])
		}
		u := &url.URL{Scheme: "vmess", Host: net.JoinHostPort(str("add"), str("port")), User: url.User(str("id")), Fragment: str("ps")}
		q := url.Values{}
		q.Set("type", str("net"))
		q.Set("host", str("host"))
		q.Set("path", str("path"))
		q.Set("sni", str("sni"))
		q.Set("security", str("tls"))
		q.Set("encryption", str("scy"))
		q.Set("alterId", str("aid"))
		u.RawQuery = q.Encode()
		return parseXrayURL(u)
	}
	u, e := url.Parse(link)
	if e != nil {
		return nil, e
	}
	return parseXrayURL(u)
}
func parseXrayURL(u *url.URL) (*NodeConfig, error) {
	if u.Scheme != "vless" && u.Scheme != "trojan" && u.Scheme != "vmess" {
		return nil, fmt.Errorf("поддерживаются VLESS, VMess и Trojan")
	}
	port, e := strconv.Atoi(u.Port())
	if e != nil || port < 1 || port > 65535 || u.Hostname() == "" || u.User == nil || u.User.Username() == "" {
		return nil, fmt.Errorf("ссылка Xray должна содержать ключ, сервер и порт")
	}
	q := u.Query()
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	if network == "splithttp" {
		network = "xhttp"
	}
	stream := map[string]any{"network": network}
	security := q.Get("security")
	if security == "" {
		if u.Scheme == "trojan" {
			security = "tls"
		} else {
			security = "none"
		}
	}
	stream["security"] = security
	sni := q.Get("sni")
	if sni == "" {
		sni = u.Hostname()
	}
	switch security {
	case "none":
	case "tls":
		tls := map[string]any{"serverName": sni, "allowInsecure": false}
		if fp := q.Get("fp"); fp != "" {
			tls["fingerprint"] = fp
		}
		if a := q.Get("alpn"); a != "" {
			tls["alpn"] = strings.Split(a, ",")
		}
		stream["tlsSettings"] = tls
	case "reality":
		if q.Get("pbk") == "" {
			return nil, fmt.Errorf("Reality: отсутствует pbk")
		}
		fp := q.Get("fp")
		if fp == "" {
			fp = "chrome"
		}
		stream["realitySettings"] = map[string]any{"serverName": sni, "fingerprint": fp, "publicKey": q.Get("pbk"), "shortId": q.Get("sid"), "spiderX": q.Get("spx")}
	default:
		return nil, fmt.Errorf("неподдерживаемая защита Xray: %s", security)
	}
	switch network {
	case "tcp", "raw":
	case "xhttp":
		mode := q.Get("mode")
		if mode == "" {
			mode = "auto"
		}
		x := map[string]any{"host": q.Get("host"), "path": q.Get("path"), "mode": mode}
		if extra := q.Get("extra"); extra != "" {
			var value map[string]any
			if e := json.Unmarshal([]byte(extra), &value); e != nil {
				return nil, fmt.Errorf("xhttp: некорректный extra")
			}
			x["extra"] = value
		}
		stream["xhttpSettings"] = x
	case "ws":
		stream["wsSettings"] = map[string]any{"path": q.Get("path"), "headers": map[string]string{"Host": q.Get("host")}}
	case "grpc":
		stream["grpcSettings"] = map[string]any{"serviceName": q.Get("serviceName"), "multiMode": q.Get("mode") == "multi"}
	default:
		return nil, fmt.Errorf("неподдерживаемый транспорт Xray: %s", network)
	}
	user := map[string]any{"id": u.User.Username()}
	if u.Scheme == "vless" {
		enc := q.Get("encryption")
		if enc == "" {
			enc = "none"
		}
		user["encryption"] = enc
		if q.Get("flow") != "" {
			user["flow"] = q.Get("flow")
		}
	}
	if u.Scheme == "vmess" {
		scy := q.Get("encryption")
		if scy == "" {
			scy = "auto"
		}
		user["security"] = scy
		aid, _ := strconv.Atoi(q.Get("alterId"))
		user["alterId"] = aid
	}
	settings := map[string]any{"vnext": []any{map[string]any{"address": u.Hostname(), "port": port, "users": []any{user}}}}
	if u.Scheme == "trojan" {
		settings = map[string]any{"servers": []any{map[string]any{"address": u.Hostname(), "port": port, "password": u.User.Username()}}}
	}
	outbound := map[string]any{"tag": "proxy", "protocol": u.Scheme, "settings": settings, "streamSettings": stream}
	raw, _ := json.Marshal(outbound)
	name := u.Fragment
	if name == "" {
		name = "Xray · " + u.Hostname()
	}
	return &NodeConfig{Protocol: "xray", Name: name, ServerAddress: u.Host, Key: u.User.Username(), EngineConfig: string(raw), Enabled: true}, nil
}

func writeEngineRuntimeConfigs(store *Store, cfg *AppConfig, node NodeConfig) (string, string, int, error) {
	port := node.LocalPort
	if port <= 0 {
		port = cfg.Core.LocalPort
	}
	dir := filepath.Join(store.RuntimeDir(), "configs")
	if e := ensureDir(dir); e != nil {
		return "", "", 0, e
	}
	var engine any
	if e := json.Unmarshal([]byte(node.EngineConfig), &engine); e != nil {
		return "", "", 0, fmt.Errorf("конфигурация ядра повреждена")
	}
	if protocolOf(node) == "xray" {
		rules := []any{}
		if cfg.Routing.ProxyMode == "direct" {
			rules = append(rules, map[string]any{"type": "field", "network": "tcp,udp", "outboundTag": "direct"})
		}
		if cfg.Routing.CustomRulesEnabled {
			domains := []string{}
			for _, line := range strings.Split(cfg.Routing.CustomRules, "\n") {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "DOMAIN-SUFFIX,")
				line = strings.TrimSuffix(line, ",DIRECT")
				if line != "" {
					domains = append(domains, "domain:"+line)
				}
			}
			if len(domains) > 0 {
				rules = append(rules, map[string]any{"type": "field", "domain": domains, "outboundTag": "direct"})
			}
		}
		engine = map[string]any{"log": map[string]any{"loglevel": "warning"}, "inbounds": []any{map[string]any{"listen": "127.0.0.1", "port": port, "protocol": "socks", "settings": map[string]any{"auth": "noauth", "udp": true}}}, "outbounds": []any{engine, map[string]any{"tag": "direct", "protocol": "freedom"}}, "routing": map[string]any{"domainStrategy": "AsIs", "rules": rules}}
	}
	configPath := filepath.Join(dir, "engine.json")
	if e := writeJSONFile(configPath, engine); e != nil {
		return "", "", 0, e
	}
	hevPath := filepath.Join(dir, "hev.yml")
	b, e := yaml.Marshal(buildHevConfig(cfg, port))
	if e != nil {
		return "", "", 0, e
	}
	e = os.WriteFile(hevPath, b, 0600)
	return configPath, hevPath, port, e
}
