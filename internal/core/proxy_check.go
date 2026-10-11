package core

import (
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"time"
)

// CheckProxyURL verifies actual traffic through the running core, never a
// direct fallback. The optional URL supports isolated integration tests.
func (b *Backend) CheckProxyURL(target string) error {
	if target == "" {
		target = "https://www.gstatic.com/generate_204"
	}
	u, e := url.Parse(target)
	if e != nil || u.User != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("некорректный адрес проверки")
	}
	b.mu.RLock()
	port := b.runningLocalPort
	b.mu.RUnlock()
	if port <= 0 {
		return fmt.Errorf("ядро не запущено")
	}
	dialer, e := proxy.SOCKS5("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)), nil, &net.Dialer{Timeout: 10 * time.Second})
	if e != nil {
		return e
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return fmt.Errorf("контекст SOCKS недоступен")
	}
	tr := &http.Transport{DialContext: contextDialer.DialContext, TLSHandshakeTimeout: 8 * time.Second, ResponseHeaderTimeout: 8 * time.Second}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 12 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if e != nil {
		return e
	}
	response, e := client.Do(req)
	if e != nil {
		return fmt.Errorf("сервер не передаёт трафик: проверьте подключение и профиль")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("проверка соединения: HTTP %d", response.StatusCode)
	}
	return nil
}
