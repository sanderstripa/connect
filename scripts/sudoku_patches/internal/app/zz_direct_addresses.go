//go:build sudoku_patch

package app

import (
	"net"
	"sync"
	"time"
)

// Preserve real-IP associations learned while resolving DIRECT destinations.
// Browsers can reuse these addresses for QUIC or HTTPS alternative services.
var directAddressNames = struct {
	sync.Mutex
	entries map[string]map[string]time.Time
}{entries: make(map[string]map[string]time.Time)}

func rememberDirectAddresses(host string, ips []net.IP) {
	directAddressNames.Lock()
	defer directAddressNames.Unlock()
	now := time.Now()
	for ip, names := range directAddressNames.entries {
		for name, expiry := range names {
			if !expiry.After(now) {
				delete(names, name)
			}
		}
		if len(names) == 0 {
			delete(directAddressNames.entries, ip)
		}
	}
	for _, ip := range ips {
		if !usableDirectIPv4(ip) {
			continue
		}
		key := ip.String()
		if directAddressNames.entries[key] == nil {
			if len(directAddressNames.entries) >= 4096 {
				continue
			}
			directAddressNames.entries[key] = make(map[string]time.Time)
		}
		directAddressNames.entries[key][host] = now.Add(2 * time.Minute)
	}
}

func matchDirectAddress(mgrs *routeManagers, addr string, ip net.IP) (bool, string) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || net.ParseIP(host) == nil {
		return false, ""
	}
	if ip == nil {
		ip = net.ParseIP(host)
	}
	directAddressNames.Lock()
	names := make([]string, 0)
	for name, expiry := range directAddressNames.entries[ip.String()] {
		if expiry.After(time.Now()) {
			names = append(names, name)
		}
	}
	directAddressNames.Unlock()
	for _, name := range names {
		if ok, _, _ := matchRuleManager(mgrs, routeActionDirect, net.JoinHostPort(name, "443"), nil); ok {
			return true, "DIRECT/DNS-IP(" + name + ")"
		}
	}
	return false, ""
}
