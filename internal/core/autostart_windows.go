//go:build windows

package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	windowsRunKeyPath       = `Software\Microsoft\Windows\CurrentVersion\Run`
	windowsRunValueName     = "connect-app"
	windowsAutostartArgLine = "--autostart"
)

var legacyAutostartCleanup sync.Once

func setLaunchAtLogin(enabled bool) error {
	exe, err := currentExecutablePath()
	if err != nil {
		return err
	}
	if err := setWindowsRunEntry(windowsRunKeyPath, windowsRunValueName, exe, enabled); err != nil {
		return fmt.Errorf("Не удалось изменить автозапуск Windows: %w", err)
	}
	// Migration must not block saving settings or depend on Task Scheduler availability.
	legacyAutostartCleanup.Do(func() { go removeLegacyAutostartTask() })
	return nil
}

func setWindowsRunEntry(path, name, exe string, enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		err := key.DeleteValue(name)
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	command := windowsAutostartCommand(exe)
	if err := key.SetStringValue(name, command); err != nil {
		return err
	}
	actual, _, err := key.GetStringValue(name)
	if err != nil {
		return err
	}
	if actual != command {
		return errors.New("autostart verification failed")
	}
	return nil
}

func windowsAutostartCommand(exe string) string {
	return `"` + exe + `" ` + windowsAutostartArgLine
}

func removeLegacyAutostartTask() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	script := strings.Join([]string{
		"$ErrorActionPreference = 'Stop'",
		"$identity = [Security.Principal.WindowsIdentity]::GetCurrent()",
		"$taskName = 'Connect-App-' + $identity.User.Value",
		"if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) { Unregister-ScheduledTask -TaskName $taskName -Confirm:$false }",
	}, "\n")
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	applyManagedProcessSysProcAttr(cmd)
	_ = cmd.Run()
}
