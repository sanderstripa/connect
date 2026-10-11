package core

import (
	"os"
	"testing"
	"time"
)

func TestManagedProcessHelper(t *testing.T) {
	switch os.Getenv("SUDOKU_PROCESS_TEST") {
	case "wait":
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "fail":
		os.Exit(1)
	}
}

func TestManagedProcessIntentionalStopAndRestart(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	p := NewManagedProcess("test")
	for i := 0; i < 8; i++ {
		if err := p.Start(exe, []string{"-test.run=^TestManagedProcessHelper$"}, []string{"SUDOKU_PROCESS_TEST=wait"}, "", nil); err != nil {
			t.Fatal(err)
		}
		if err := p.Stop(3 * time.Second); err != nil {
			t.Fatalf("intentional stop returned error: %v", err)
		}
		if p.IsRunning() {
			t.Fatal("Stop returned before process state was cleared")
		}
	}
}

func TestManagedProcessExitCallbackCanInspectProcess(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	p := NewManagedProcess("test")
	callback := make(chan string, 1)
	if err := p.Start(exe, []string{"-test.run=^TestManagedProcessHelper$"}, []string{"SUDOKU_PROCESS_TEST=fail"}, "", func(line string) {
		_ = p.IsRunning()
		callback <- line
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case line := <-callback:
		if line == "" {
			t.Fatal("missing unexpected-exit diagnostic")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("exit callback deadlocked inspecting process")
	}
}
