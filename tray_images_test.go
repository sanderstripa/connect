package main

import (
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/core"
	"testing"
)

func TestTrayRequiresConfirmedTunnel(t *testing.T) {
	for mask := 0; mask < 16; mask++ {
		state := core.RuntimeState{Running: mask&1 != 0, CoreRunning: mask&2 != 0, TunRunning: mask&4 != 0}
		if mask&8 != 0 {
			state.RouteSetupError = "route failed"
		}
		if got := trayConnected(state); got != (mask == 7) {
			t.Errorf("state mask %d: connected=%v", mask, got)
		}
	}
}

func TestTrayConnectionLifecycle(t *testing.T) {
	states := []core.RuntimeState{
		{},
		{CoreRunning: true},
		{Running: true, CoreRunning: true, TunRunning: true},
		{Running: true, CoreRunning: true},                   // unexpected tunnel loss
		{Running: true, CoreRunning: true, TunRunning: true}, // recovery
		{}, // explicit disconnect
	}
	expected := []bool{false, false, true, false, true, false}
	for i, state := range states {
		if got := trayConnected(state); got != expected[i] {
			t.Errorf("step %d: connected=%v", i, got)
		}
	}
}
