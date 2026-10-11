//go:build linux

package main

import (
	"errors"
	"sync"

	"github.com/sanderstripa/vpn/mist/internal/tunio"
)

type edgePacketSink interface {
	WritePacket([]byte) error
}

type edgeDispatcher struct {
	dev  *tunio.Device
	mu   sync.RWMutex
	sink edgePacketSink
}

func newEdgeDispatcher(dev *tunio.Device) *edgeDispatcher {
	d := &edgeDispatcher{dev: dev}
	go d.run()
	return d
}

func (d *edgeDispatcher) register(s edgePacketSink) (func(), error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sink != nil {
		return nil, errors.New("another MIST client transport is active")
	}
	d.sink = s
	released := false
	return func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		if released {
			return
		}
		released = true
		if d.sink == s {
			d.sink = nil
		}
	}, nil
}

func (d *edgeDispatcher) run() {
	for {
		packets, err := d.dev.ReadBatch()
		if err != nil {
			return
		}
		for _, packet := range packets {
			d.mu.RLock()
			s := d.sink
			if s != nil {
				_ = s.WritePacket(packet)
			}
			d.mu.RUnlock()
		}
	}
}
