package app

import (
	"net"
	"testing"
	"time"
)

func TestDirectAddressRouting(t *testing.T) {
	directAddressNames.entries = make(map[string]map[string]time.Time)
	manager := newRuleManagerForTest(t, "DOMAIN-SUFFIX,ozon.ru,DIRECT\n")
	mgrs := &routeManagers{direct: manager}
	ips := []net.IP{net.ParseIP("185.73.193.68"), net.ParseIP("185.73.194.82")}
	rememberDirectAddresses("www.ozon.ru", ips)
	for _, ip := range ips {
		if ok, _ := matchDirectAddress(mgrs, net.JoinHostPort(ip.String(), "443"), ip); !ok {
			t.Fatal("known DIRECT address not matched", ip)
		}
	}
	if ok, _ := matchDirectAddress(mgrs, "1.1.1.1:443", nil); ok {
		t.Fatal("unrelated IP bypassed VPN")
	}
	directAddressNames.entries[ips[0].String()]["www.ozon.ru"] = time.Now().Add(-time.Second)
	if ok, _ := matchDirectAddress(mgrs, "185.73.193.68:443", nil); ok {
		t.Fatal("expired association retained")
	}
	if ok, _ := matchDirectAddress(&routeManagers{}, "185.73.194.82:443", nil); ok {
		t.Fatal("removed rule still bypasses VPN")
	}
}
