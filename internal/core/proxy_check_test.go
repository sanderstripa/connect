package core

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTrafficReadinessRetriesWithoutReconnecting(t *testing.T) {
	calls := 0
	err := retryTrafficReadiness(context.Background(), func() error {
		calls++
		if calls == 1 {
			return errors.New("adapter settling")
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("readiness did not recover: calls=%d err=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls = 0
	err = retryTrafficReadiness(ctx, func() error { calls++; cancel(); return errors.New("not ready") })
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("cancellation did not stop retry: %d %v", calls, err)
	}
}

func TestLiveProxyHealth(t *testing.T) {
	value := os.Getenv("CONNECT_HEALTH_TEST_PORT")
	if value == "" {
		t.Skip("optional read-only check through an already running local proxy")
	}
	port, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = checkProxyURL(context.Background(), "", port, 8*time.Second); err != nil {
		t.Fatal(err)
	}
}

// A real SOCKS server in the test ensures probes never silently dial directly.
func healthTestSOCKS(t *testing.T, drop bool) int {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, e := l.Accept()
			if e != nil {
				return
			}
			go func() {
				defer c.Close()
				c.SetDeadline(time.Now().Add(3 * time.Second))
				var greeting [3]byte
				if _, e := io.ReadFull(c, greeting[:]); e != nil {
					return
				}
				c.Write([]byte{5, 0})
				var h [4]byte
				if _, e := io.ReadFull(c, h[:]); e != nil {
					return
				}
				var host string
				switch h[3] {
				case 1:
					var ip [4]byte
					if _, e := io.ReadFull(c, ip[:]); e != nil {
						return
					}
					host = net.IP(ip[:]).String()
				case 3:
					var n [1]byte
					if _, e := io.ReadFull(c, n[:]); e != nil {
						return
					}
					b := make([]byte, int(n[0]))
					if _, e := io.ReadFull(c, b); e != nil {
						return
					}
					host = string(b)
				default:
					return
				}
				var p [2]byte
				if _, e := io.ReadFull(c, p[:]); e != nil {
					return
				}
				c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
				if drop {
					return
				}
				upstream, e := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(p[:])))), time.Second)
				if e != nil {
					return
				}
				defer upstream.Close()
				upstream.SetDeadline(time.Now().Add(3 * time.Second))
				go io.Copy(upstream, c)
				io.Copy(c, upstream)
			}()
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

func TestPreflightRejectsSOCKSSuccessWithoutTraffic(t *testing.T) {
	port := healthTestSOCKS(t, true)
	b := &Backend{preflightTarget: "127.0.0.1:443"}
	if e := healthCheckSOCKS5Connect(context.Background(), net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), b.preflightTarget, time.Second); e != nil {
		t.Fatal("fixture must acknowledge SOCKS", e)
	}
	if e := b.preflightTunUpstream(context.Background(), port, RoutingSettings{ProxyMode: "global"}, time.Now()); e == nil {
		t.Fatal("dead upstream accepted before TUN")
	}
	if b.state.Running || b.routeState != nil {
		t.Fatal("failed probe changed VPN state")
	}
}

func TestProxyHealthDoesNotBypassSOCKSOrDisableTLSVerification(t *testing.T) {
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer target.Close()
	if e := checkProxyURL(context.Background(), target.URL, healthTestSOCKS(t, false), time.Second); e == nil {
		t.Fatal("untrusted TLS certificate accepted")
	}
	if e := checkProxyURL(context.Background(), target.URL, healthTestSOCKS(t, true), time.Second); e == nil {
		t.Fatal("direct fallback bypassed failed SOCKS")
	}
}

func TestProxyHealthFallbackAndChallenge(t *testing.T) {
	challenge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cf-Mitigated", "challenge")
		w.WriteHeader(403)
	}))
	defer challenge.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer good.Close()
	port := healthTestSOCKS(t, false)
	if e := checkProxyURL(context.Background(), challenge.URL, port, time.Second); e == nil || !strings.Contains(e.Error(), "Cloudflare") {
		t.Fatalf("missing site challenge diagnosis: %v", e)
	}
	if e := checkProxyTargets(context.Background(), []string{challenge.URL, good.URL}, port, time.Second); e != nil {
		t.Fatal("one blocked endpoint rejected working proxy", e)
	}
	if e := checkProxyTargets(context.Background(), []string{challenge.URL, challenge.URL}, port, time.Second); e == nil {
		t.Fatal("failed probes accepted")
	}
}

func TestProxyHealthRejectsRedirectAndTruncatedBody(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/ok", 302)
			return
		}
		w.Header().Set("Content-Length", "100")
		w.Write([]byte("short"))
	}))
	defer target.Close()
	port := healthTestSOCKS(t, false)
	for _, suffix := range []string{"/redirect", "/truncated"} {
		if e := checkProxyURL(context.Background(), target.URL+suffix, port, time.Second); e == nil {
			t.Fatal("invalid health response accepted", suffix)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := checkProxyURL(ctx, target.URL, port, time.Second); e == nil {
		t.Fatal("cancelled probe accepted")
	}
}
