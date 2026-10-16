//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestUpdateHandoffWaitsAndLaunchesAnIsolatedInstaller(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "path with spaces ' and unicode тест")
	if e := os.MkdirAll(dir, 0700); e != nil {
		t.Fatal(e)
	}
	source := filepath.Join(dir, "probe.go")
	file := filepath.Join(dir, "Connect-Setup.exe")
	marker := filepath.Join(dir, "launched.txt")
	code := `package main
import("os";"time")
func main(){if len(os.Args)>1{time.Sleep(time.Second);return};os.WriteFile(os.Getenv("CONNECT_INSTALLER_TEST_RESULT"),[]byte("OK"),0600)}`
	os.WriteFile(source, []byte(code), 0600)
	build := exec.Command("go", "build", "-ldflags=-H=windowsgui", "-o", file, source)
	build.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatalf("probe build: %s %v", out, e)
	}
	parent := exec.Command(file, "parent")
	parent.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if e := parent.Start(); e != nil {
		t.Fatal(e)
	}
	defer parent.Wait()
	launch := installerLaunchCommand(file, parent.Process.Pid)
	launch.Env = append(os.Environ(), "CONNECT_INSTALLER_TEST_RESULT="+marker)
	if e := launch.Start(); e != nil {
		t.Fatal(e)
	}
	defer launch.Wait()
	time.Sleep(200 * time.Millisecond)
	if _, e := os.Stat(marker); e == nil {
		t.Fatal("installer launched before parent exited")
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if b, e := os.ReadFile(marker); e == nil && string(b) == "OK" {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("installer was not launched")
}
