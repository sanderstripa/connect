//go:build windows

package core

import (
	"context"
	"testing"
)

func TestWindowsExceptionsKeepMapDNSDomainAssociation(t *testing.T) {
	cfg := &AppConfig{Routing: RoutingSettings{ProxyMode: "pac", CustomRulesEnabled: true, CustomRules: "DOMAIN-SUFFIX,hh.ru,DIRECT\nDOMAIN-SUFFIX,ozon.ru,DIRECT"}, Tun: TunSettings{MapDNSEnabled: true, MapDNSAddress: "198.18.0.1", MapDNSPort: 53}}
	// No backend/network state is needed: explicit domain mode must never
	// create the DNS proxy that resolves excluded names to real IPs.
	b := &Backend{}
	dns, err := b.prepareTunDNSRuntime(context.Background(), cfg, 1080)
	if err != nil {
		t.Fatal(err)
	}
	if dns.proxy != nil || dns.systemDNSAddress != "198.18.0.1" || dns.healthAddr != "198.18.0.1:53" {
		t.Fatalf("domain association lost: %#v", dns)
	}
}
