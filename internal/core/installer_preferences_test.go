package core

import (
	"encoding/json"
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/installprefs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInstallerPreferencesFreshAndUpdate(t *testing.T) {
	for _, existing := range []bool{false, true} {
		root := t.TempDir()
		s := &Store{rootDir: root, configPath: filepath.Join(root, "config.json"), runtimeDir: filepath.Join(root, "runtime"), logDir: filepath.Join(root, "logs")}
		cfg, err := s.Load()
		if err != nil {
			t.Fatal(err)
		}
		if existing {
			cfg.UI.Language = "ru"
			cfg.UI.Theme = "light"
			cfg.Routing.CustomRules = "keep rules"
			cfg.Core.AutoStart = true
			cfg.ActiveNodeID = "keep node"
			if err = s.Save(cfg); err != nil {
				t.Fatal(err)
			}
		}
		before := *cfg
		path := filepath.Join(t.TempDir(), installprefs.File)
		write := func(id, language, theme string) {
			raw, _ := json.Marshal(installprefs.Preferences{id, language, theme})
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
		write("install-1", "en", "dark")
		if err = s.applyInstallerPreferences(cfg, path); err != nil {
			t.Fatal(err)
		}
		loaded, err := s.Load()
		if err != nil {
			t.Fatal(err)
		}
		if loaded.UI.Language != "en" || loaded.UI.Theme != "dark" {
			t.Fatal(loaded.UI)
		}
		expected := before
		expected.UI.Language = "en"
		expected.UI.Theme = "dark"
		if !reflect.DeepEqual(*loaded, expected) {
			t.Fatal("unrelated settings changed")
		}
		for _, language := range []string{"ru", "en"} {
			for _, theme := range []string{"light", "dark"} {
				write(language+theme, language, theme)
				if err = s.applyInstallerPreferences(loaded, path); err != nil {
					t.Fatal(err)
				}
				persisted, e := s.Load()
				if e != nil || persisted.UI.Language != language || persisted.UI.Theme != theme {
					t.Fatal("selection not persisted", e)
				}
			}
		}
		write("install-1", "en", "dark")
		if err = s.applyInstallerPreferences(loaded, path); err != nil {
			t.Fatal(err)
		}
		loaded.UI.Language = "ru"
		loaded.UI.Theme = "light"
		if err = s.Save(loaded); err != nil {
			t.Fatal(err)
		}
		if err = s.applyInstallerPreferences(loaded, path); err != nil {
			t.Fatal(err)
		}
		if loaded.UI.Language != "ru" || loaded.UI.Theme != "light" {
			t.Fatal("same installation reapplied")
		}
		write("install-2", "en", "dark")
		if err = s.applyInstallerPreferences(loaded, path); err != nil {
			t.Fatal(err)
		}
		if loaded.UI.Language != "en" || loaded.UI.Theme != "dark" {
			t.Fatal("reinstall not applied")
		}
		write("install-3", "invalid", "dark")
		if err = s.applyInstallerPreferences(loaded, path); err == nil {
			t.Fatal("invalid preferences accepted")
		}
		unchanged := *loaded
		if err = s.applyInstallerPreferences(loaded, path+"-missing"); err != nil || !reflect.DeepEqual(*loaded, unchanged) {
			t.Fatal("missing handoff changed settings", err)
		}
	}
}
