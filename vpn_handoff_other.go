//go:build !windows

package main

import (
	"io"
	"time"
)

type inactiveVPNPeer struct{}

func (inactiveVPNPeer) Close() error { return nil }

func lockVPNTransition() (func(), error) { return func() {}, nil }

func serveVPNPeer(string, func(string) (vpnPeerState, error)) (io.Closer, error) {
	return inactiveVPNPeer{}, nil
}

func contactVPNPeer(string, string, time.Duration) (vpnPeerState, error) {
	return vpnPeerState{}, nil
}
