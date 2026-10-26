package core

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// CheckProxyURL verifies actual traffic through the running core, never a
// direct fallback. The optional URL supports isolated integration tests.
func (b *Backend) CheckProxyURL(target string) error {
	b.mu.RLock()
	port := b.runningLocalPort
	b.mu.RUnlock()
	return checkProxyURL(context.Background(), target, port, 12*time.Second)
}

func checkProxyURL(parent context.Context, target string, port int, timeout time.Duration) error {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	if target == "" {
		// A SOCKS CONNECT reply alone does not prove the upstream carries bytes.
		// Probe independent HTTPS endpoints, without bypassing SOCKS or weakening
		// certificate verification. One provider being blocked is not a dead VPN.
		return retryTrafficReadiness(ctx, func() error {
			return checkProxyTargets(ctx, []string{"https://www.gstatic.com/generate_204", "https://cp.cloudflare.com/generate_204"}, port, timeout)
		})
	}
	return checkSingleProxyURL(ctx, target, port, timeout)
}

// Let a new core or adapter settle without tearing down and recreating the
// user's connection. Bound both retries and the entire operation by its context.
func retryTrafficReadiness(ctx context.Context, probe func() error) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if last = probe(); last == nil {
			return nil
		}
		if attempt < 2 {
			timer := time.NewTimer(time.Duration(attempt+1) * 250 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return last
}

func checkProxyTargets(ctx context.Context, targets []string, port int, timeout time.Duration) error {
	return checkTrafficTargets(ctx, targets, func(ctx context.Context, target string) error { return checkSingleProxyURL(ctx, target, port, timeout) })
}

func checkTrafficTargets(ctx context.Context, targets []string, probe func(context.Context, string) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan error, len(targets))
	for _, target := range targets {
		go func(target string) { results <- probe(ctx, target) }(target)
	}
	var failures []error
	for range targets {
		select {
		case <-ctx.Done():
			return fmt.Errorf("проверка HTTPS через VPN: %w", ctx.Err())
		case err := <-results:
			if err == nil {
				return nil
			}
			failures = append(failures, err)
		}
	}
	return fmt.Errorf("VPN не передаёт HTTPS-трафик; проверьте профиль и совместимость ядра с сервером: %w", errors.Join(failures...))
}

func checkSingleProxyURL(parent context.Context, target string, port int, timeout time.Duration) error {
	u, e := url.Parse(target)
	if e != nil || u.User != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("некорректный адрес проверки")
	}
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
	return checkHTTPResponse(parent, target, tr, timeout)
}

func checkHTTPResponse(parent context.Context, target string, tr *http.Transport, timeout time.Duration) error {
	defer tr.CloseIdleConnections()
	u, e := url.Parse(target)
	if e != nil || u.User != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("некорректный адрес проверки")
	}
	client := &http.Client{Transport: tr, Timeout: timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if e != nil {
		return e
	}
	response, e := client.Do(req)
	if e != nil {
		return fmt.Errorf("HTTPS %s: %w", u.Hostname(), e)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode == http.StatusForbidden && response.Header.Get("Cf-Mitigated") == "challenge" {
			return fmt.Errorf("%s: HTTP 403 — Cloudflare требует проверку браузера; VPN достиг сайта", u.Hostname())
		}
		return fmt.Errorf("проверка соединения: HTTP %d", response.StatusCode)
	}
	if _, e := io.Copy(io.Discard, io.LimitReader(response.Body, 1024)); e != nil {
		return fmt.Errorf("HTTPS %s: ответ оборвался: %w", u.Hostname(), e)
	}
	return nil
}
