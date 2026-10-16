//go:build windows

package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

func launchUpdateInstaller(file string) error {
	root, e := os.UserConfigDir()
	if e != nil {
		return e
	}
	allowed := filepath.Join(root, "connect-components", "app")
	rel, e := filepath.Rel(allowed, file)
	if e != nil || strings.HasPrefix(rel, "..") || filepath.Base(file) != "Connect-Setup.exe" {
		return fmt.Errorf("invalid update installer")
	}
	return installerLaunchCommand(file, os.Getpid()).Start()
}
func installerLaunchCommand(file string, parentPID int) *exec.Cmd {
	// Wait for this UI process and its network helper to finish before launching
	// the normal elevated installer. No arbitrary command or URL comes from UI.
	script := fmt.Sprintf("$ErrorActionPreference='Stop'; Wait-Process -Id %d -ErrorAction SilentlyContinue; Start-Sleep -Seconds 2; Start-Process -FilePath '%s'", parentPID, strings.ReplaceAll(file, "'", "''"))
	units := utf16.Encode([]rune(script))
	buf := make([]byte, len(units)*2)
	for i, v := range units {
		binary.LittleEndian.PutUint16(buf[i*2:], v)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-EncodedCommand", base64.StdEncoding.EncodeToString(buf))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
