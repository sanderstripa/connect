package core

import (
	"path/filepath"
	"testing"
)

func TestSavedSettingsSurviveReloadAndReplacement(t *testing.T) {
	root := t.TempDir()
	store := &Store{rootDir: root, configPath: filepath.Join(root, "config.json"), runtimeDir: filepath.Join(root, "runtime"), logDir: filepath.Join(root, "logs")}
	cfg := DefaultConfig(store.runtimeDir)
	b := &Backend{store: store, cfg: cfg, pfMgr: newPortForwardManager(func(string) {})}
	next := *cloneConfig(cfg)
	next.UI.Theme = "dark"
	next.Core.AutoStart = true
	next.Routing.ProxyMode = "pac"
	next.Routing.CustomRulesEnabled = true
	next.Routing.CustomRules = "DOMAIN-SUFFIX,ozon.ru,DIRECT"
	if err := b.SaveConfig(next); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.UI.Theme != "dark" || !loaded.Core.AutoStart || loaded.Routing.CustomRules != next.Routing.CustomRules {
		t.Fatal("settings not persisted")
	}
	next.UI.Theme = "light"
	if err := b.SaveConfig(next); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load()
	if err != nil || loaded.UI.Theme != "light" {
		t.Fatal("replacement did not persist", err)
	}
	entries, _ := filepath.Glob(filepath.Join(root, ".config-*.tmp"))
	if len(entries) != 0 {
		t.Fatal("temporary config files remain")
	}
}
