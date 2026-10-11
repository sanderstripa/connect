package core

import (
	"reflect"
	"testing"
)

func TestDeviceVPNNormalizesExistingWindowsProfile(t *testing.T) {
	c := &AppConfig{Version: 6, Tun: TunSettings{Enabled: false}, Routing: RoutingSettings{ProxyMode: "global"}}
	normalizeConfigForOS(c, t.TempDir(), "windows")
	if !c.Tun.Enabled {
		t.Fatal("existing profile must use device VPN")
	}
}
func TestEmptySplitListKeepsModeWithoutDefaultRules(t *testing.T) {
	c := &AppConfig{Version: 7, Routing: RoutingSettings{ProxyMode: "pac", CustomRulesEnabled: true}}
	normalizeConfigForOS(c, t.TempDir(), "windows")
	if c.Routing.ProxyMode != "pac" || len(c.Routing.RuleURLs) != 0 {
		t.Fatal("empty exception list must remain selected without external rules")
	}
	r, e := buildSudokuClientConfig(c, NodeConfig{ServerAddress: "example.com:443", Key: "test"}, "", false)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(r.RuleURLs, []string{"global"}) {
		t.Fatal(r.RuleURLs)
	}
}
func TestSplitListUsesOnlyExplicitExceptions(t *testing.T) {
	c := &AppConfig{Version: 7, Routing: RoutingSettings{ProxyMode: "pac", CustomRulesEnabled: true, CustomRules: "DOMAIN-SUFFIX,hh.ru,DIRECT"}}
	normalizeConfigForOS(c, t.TempDir(), "windows")
	r, e := buildSudokuClientConfig(c, NodeConfig{ServerAddress: "example.com:443", Key: "test"}, "http://127.0.0.1:12345/custom", false)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(r.RuleURLs, []string{"http://127.0.0.1:12345/custom"}) {
		t.Fatal(r.RuleURLs)
	}
}
