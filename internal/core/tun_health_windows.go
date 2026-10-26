//go:build windows

package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// Pin the probe socket to the TUN adapter. The physical default route and an
// environment/system proxy must never hide a failed desktop VPN dataplane.
func checkTunHTTPS(ctx context.Context, ifIndex int) error {
	if ifIndex <= 0 {
		return fmt.Errorf("TUN adapter is unavailable for HTTPS verification")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return retryTrafficReadiness(ctx, func() error {
		return checkTrafficTargets(ctx, []string{"https://www.gstatic.com/generate_204", "https://cp.cloudflare.com/generate_204"}, func(ctx context.Context, target string) error {
			dialer := &net.Dialer{Timeout: 4 * time.Second, Control: func(_ string, _ string, raw syscall.RawConn) error {
				var bindErr error
				if err := raw.Control(func(fd uintptr) { bindErr = winBind4(syscall.Handle(fd), ifIndex) }); err != nil {
					return err
				}
				return bindErr
			}}
			tr := &http.Transport{DialContext: func(ctx context.Context, _ string, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp4", addr)
			}, TLSHandshakeTimeout: 4 * time.Second, ResponseHeaderTimeout: 4 * time.Second}
			return checkHTTPResponse(ctx, target, tr, 8*time.Second)
		})
	})
}
