//go:build windows

package core

import (
	"context"
	"os"
	"strconv"
	"testing"
)

func TestTunnelHealthRequiresAdapter(t *testing.T) {
	if err := checkTunHTTPS(context.Background(), 0); err == nil {
		t.Fatal("TUN health check allowed an unspecified interface")
	}
}
func TestLiveTunnelHealth(t *testing.T) {
	value := os.Getenv("CONNECT_HEALTH_TUN_INDEX")
	if value == "" {
		t.Skip("optional read-only probe through an already running TUN")
	}
	idx, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = checkTunHTTPS(context.Background(), idx); err != nil {
		t.Fatal(err)
	}
}
