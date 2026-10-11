package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Fetches only the subscription the user explicitly pastes; never executes YAML.
func ParseImportLink(link string) (*NodeConfig, error) {
	if strings.HasPrefix(link, "vless://") || strings.HasPrefix(link, "trojan://") || strings.HasPrefix(link, "vmess://") || strings.HasPrefix(link, "mist://") || strings.HasPrefix(link, "{") { return ParseEngineLink(link) }
	if strings.HasPrefix(link, "sudoku://") {
		return ParseShortLink(link)
	}
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return nil, fmt.Errorf("Вставьте sudoku:// или HTTP(S)-ссылку подписки из Sudoku UI")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("Некорректная ссылка подписки")
	}
	req.Header.Set("User-Agent", "Sudoku-App/0.1")
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 5 || r.URL.User != nil || (r.URL.Scheme != "https" && r.URL.Scheme != "http") || (via[0].URL.Scheme == "https" && r.URL.Scheme != "https") {
			return fmt.Errorf("Недопустимое перенаправление")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Не удалось загрузить подписку. Проверьте ссылку и соединение")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Подписка недоступна (HTTP %d)", resp.StatusCode)
	}
	const limit = 1024 * 1024
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(raw) > limit {
		return nil, fmt.Errorf("Не удалось прочитать подписку: ограничение 1 МБ")
	}
	node, err := parseSubscription(raw)
	if err != nil {
		return nil, err
	}
	node.SourceURL = link
	return node, nil
}

func parseSubscription(raw []byte) (*NodeConfig, error) {
	var doc struct {
		Proxies []struct {
			Name        string `yaml:"name"`
			Type        string `yaml:"type"`
			Server      string `yaml:"server"`
			Port        int    `yaml:"port"`
			Key         string `yaml:"key"`
			AEAD        string `yaml:"aead-method"`
			ASCII       string `yaml:"table-type"`
			PaddingMin  int    `yaml:"padding-min"`
			PaddingMax  int    `yaml:"padding-max"`
			CustomTable string `yaml:"custom-table"`
			Multiplex   string `yaml:"multiplex"`
			Pure        bool   `yaml:"enable-pure-downlink"`
			HTTP        struct {
				Disable bool   `yaml:"disable"`
				Mode    string `yaml:"mode"`
				TLS     bool   `yaml:"tls"`
				Host    string `yaml:"host"`
				Path    string `yaml:"path-root"`
			} `yaml:"httpmask"`
		} `yaml:"proxies"`
	}
	if yaml.Unmarshal(raw, &doc) != nil || len(doc.Proxies) != 1 {
		return nil, fmt.Errorf("Ожидается подписка Sudoku UI с одним сервером")
	}
	p := doc.Proxies[0]
	if p.Type != "sudoku" || strings.TrimSpace(p.Server) == "" || p.Port < 1 || p.Port > 65535 || strings.TrimSpace(p.Key) == "" || p.PaddingMin < 0 || p.PaddingMax < p.PaddingMin {
		return nil, fmt.Errorf("Подписка содержит некорректный сервер Sudoku")
	}
	n := &NodeConfig{ID: newID("node_"), Name: p.Name, ServerAddress: net.JoinHostPort(p.Server, strconv.Itoa(p.Port)), Key: p.Key, AEAD: p.AEAD, ASCII: decodeASCII(p.ASCII), PaddingMin: p.PaddingMin, PaddingMax: p.PaddingMax, CustomTable: p.CustomTable, Multiplex: p.Multiplex, EnablePureDownlink: p.Pure, LocalPort: 1080, Enabled: true, HTTPMask: HTTPMaskSettings{Disable: p.HTTP.Disable, Mode: p.HTTP.Mode, TLS: p.HTTP.TLS, Host: p.HTTP.Host, PathRoot: p.HTTP.Path}}
	if n.Name == "" {
		n.Name = p.Server
	}
	return n, nil
}
