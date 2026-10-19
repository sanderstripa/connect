package core

import (
	"encoding/json"
	"fmt"
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/installprefs"
	"os"
	"path/filepath"
)

func (s *Store) applyInstallerPreferences(cfg *AppConfig, path string) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read installer preferences: %w", err)
	}
	var p installprefs.Preferences
	if err = json.Unmarshal(raw, &p); err != nil || !p.Valid() {
		return fmt.Errorf("invalid installer preferences")
	}
	receipt := filepath.Join(s.rootDir, ".installer-preferences-id")
	previous, err := os.ReadFile(receipt)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if string(previous) == p.ID {
		return nil
	}
	cfg.UI.Language, cfg.UI.Theme = p.Language, p.Theme
	if err = s.Save(cfg); err != nil {
		return err
	}
	// Receipt follows the atomic configuration save, so an interrupted first launch retries safely.
	return os.WriteFile(receipt, []byte(p.ID), 0600)
}
