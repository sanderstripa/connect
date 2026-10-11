package app

import (
	"github.com/SUDOKU-ASCII/sudoku/internal/config"
	"testing"
)

func TestExplicitExceptionsKeepChatGPTInVPN(t *testing.T) {
	manager := newRuleManagerForTest(t, "DOMAIN-SUFFIX,hh.ru,DIRECT\nDOMAIN-SUFFIX,avito.ru,DIRECT\nDOMAIN-SUFFIX,ozon.ru,DIRECT\n")
	managers := &routeManagers{direct: manager}
	cfg := &config.Config{ProxyMode: "pac"}
	for _, c := range []struct{ host, action string }{{"hh.ru:443", routeActionDirect}, {"ekaterinburg.hh.ru:443", routeActionDirect}, {"api.hh.ru:443", routeActionDirect}, {"avito.ru:443", routeActionDirect}, {"ozon.ru:443", routeActionDirect}, {"www.ozon.ru:443", routeActionDirect}, {"chatgpt.com:443", routeActionProxy}, {"ws.chatgpt.com:443", routeActionProxy}, {"auth.openai.com:443", routeActionProxy}, {"cdn.oaistatic.com:443", routeActionProxy}, {"nothh.ru:443", routeActionProxy}} {
		got := decideRoute(cfg, managers, c.host, nil)
		if got.action != c.action {
			t.Fatalf("%s: got %s, want %s", c.host, got.action, c.action)
		}
	}
}
