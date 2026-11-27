//go:build windows

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

//go:embed extension
var chromeExtension embed.FS

const chromeRegistration = `Software\Google\Chrome\NativeMessagingHosts\com.sanderstripa.connect`

func (a *App) PrepareChromeExtension() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	chrome, err := findChrome()
	if err != nil {
		return err
	}
	folder, err := chromeExtensionFolder(executable)
	if err != nil {
		return err
	}

	copyPath := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive",
		"-Command", "$ErrorActionPreference='Stop'; Set-Clipboard -Value $env:CONNECT_EXTENSION_FOLDER")
	copyPath.Env = append(os.Environ(), "CONNECT_EXTENSION_FOLDER="+folder)
	copyPath.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := copyPath.CombinedOutput(); err != nil {
		return fmt.Errorf("копирование папки расширения: %w: %s", err, strings.TrimSpace(string(output)))
	}
	cmd := exec.Command(chrome, "chrome://extensions/")
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func chromeExtensionFolder(executable string) (string, error) {
	root := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(root) {
		return "", errors.New("LOCALAPPDATA is unavailable")
	}
	userFolder := filepath.Join(root, "Connect", "ChromeExtension")
	installedFolder := filepath.Join(filepath.Dir(executable), "ChromeExtension")
	// Keep the original folder of an extension already loaded in Chrome.
	if _, err := os.Stat(filepath.Join(userFolder, "manifest.json")); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Stat(filepath.Join(installedFolder, "manifest.json")); err == nil {
			return installedFolder, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	// This path is also available when the installer omitted the component.
	return userFolder, prepareUserExtension(userFolder, executable)
}

func findChrome() (string, error) {
	for _, root := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		base := os.Getenv(root)
		if !filepath.IsAbs(base) {
			continue
		}
		candidate := filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("Google Chrome не найден")
}

func prepareUserExtension(folder, executable string) error {
	if err := fs.WalkDir(chromeExtension, "extension", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destination := filepath.Join(folder, filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(path, "extension"), "/")))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		content, err := chromeExtension.ReadFile(path)
		if err != nil {
			return err
		}
		return writeChromeFile(destination, content)
	}); err != nil {
		return err
	}
	manifest, err := json.Marshal(map[string]any{
		"name": "com.sanderstripa.connect", "description": "Connect browser transport helper",
		"path": executable, "type": "stdio", "allowed_origins": []string{chromeOrigin},
	})
	if err != nil {
		return err
	}
	host := filepath.Join(filepath.Dir(folder), "chrome-native-host.json")
	if err := writeChromeFile(host, manifest); err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, chromeRegistration, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue("", host)
}

func writeChromeFile(destination string, content []byte) error {
	current, err := os.ReadFile(destination)
	if err == nil && bytes.Equal(current, content) {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(destination), ".connect-chrome-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(content)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), destination)
}
