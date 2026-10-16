package main

import (
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/updates"
	"testing"
)

func TestHeadlessHelperOnlyStagesApplicationUpdate(t *testing.T) {
	a := &App{}
	result, e := a.finishUpdate(updates.Result{Version: "v2", Installer: "must-not-be-launched.exe"}, nil)
	if e != nil || result.Installer != "must-not-be-launched.exe" {
		t.Fatal("headless helper attempted to launch setup", e)
	}
}
