package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun/netstack"
	"io"
	"net"
	"sync"
	"time"
)

func readSOCKSAddress(r io.Reader) (string, []byte, error) {
	var t [1]byte
	if _, e := io.ReadFull(r, t[:]); e != nil {
		return "", nil, e
	}
	raw := []byte{t[0]}
	var host string
	switch t[0] {
	case 1:
		b := make([]byte, 4)
		if _, e := io.ReadFull(r, b); e != nil {
			return "", nil, e
		}
		raw = append(raw, b...)
		host = net.IP(b).String()
	case 4:
		b := make([]byte, 16)
		if _, e := io.ReadFull(r, b); e != nil {
			return "", nil, e
		}
		raw = append(raw, b...)
		host = net.IP(b).String()
	case 3:
		var l [1]byte
		if _, e := io.ReadFull(r, l[:]); e != nil {
			return "", nil, e
		}
		b := make([]byte, int(l[0]))
		if _, e := io.ReadFull(r, b); e != nil {
			return "", nil, e
		}
		raw = append(raw, l[0])
		raw = append(raw, b...)
		host = string(b)
	default:
		return "", nil, fmt.Errorf("bad address")
	}
	var p [2]byte
	if _, e := io.ReadFull(r, p[:]); e != nil {
		return "", nil, e
	}
	raw = append(raw, p[:]...)
	return net.JoinHostPort(host, fmt.Sprint(binary.BigEndian.Uint16(p[:]))), raw, nil
}
func serveConnectSOCKS(ctx context.Context, c net.Conn, stack *netstack.Net) {
	defer c.Close()
	c.SetDeadline(time.Now().Add(15 * time.Second))
	var greeting [2]byte
	if _, e := io.ReadFull(c, greeting[:]); e != nil || greeting[0] != 5 {
		return
	}
	methods := make([]byte, int(greeting[1]))
	if _, e := io.ReadFull(c, methods); e != nil {
		return
	}
	ok := false
	for _, m := range methods {
		if m == 0 {
			ok = true
		}
	}
	if !ok {
		c.Write([]byte{5, 255})
		return
	}
	c.Write([]byte{5, 0})
	var req [3]byte
	if _, e := io.ReadFull(c, req[:]); e != nil || req[0] != 5 || req[2] != 0 {
		return
	}
	_, _, e := handleConnectSOCKSRequest(c, req, ctx, stack)
	if e != nil {
		return
	}
}
func handleConnectSOCKSRequest(c net.Conn, req [3]byte, ctx context.Context, stack *netstack.Net) (string, []byte, error) {
	address, raw, e := readSOCKSAddress(c)
	if e != nil {
		return "", nil, e
	}
	if req[1] == 3 {
		return "", nil, serveConnectUDP(ctx, c, stack)
	}
	if req[1] != 1 {
		c.Write([]byte{5, 7, 0, 1, 0, 0, 0, 0, 0, 0})
		return "", nil, fmt.Errorf("unsupported command")
	}
	dialCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	remote, e := stack.DialContext(dialCtx, "tcp", address)
	cancel()
	if e != nil {
		c.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
		return "", nil, e
	}
	defer remote.Close()
	c.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
	c.SetDeadline(time.Time{})
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			c.Close()
			remote.Close()
		case <-done:
		}
	}()
	go func() { io.Copy(remote, c); remote.Close() }()
	io.Copy(c, remote)
	close(done)
	return address, raw, nil
}
func serveConnectUDP(ctx context.Context, c net.Conn, stack *netstack.Net) error {
	udp, e := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if e != nil {
		return e
	}
	defer udp.Close()
	port := udp.LocalAddr().(*net.UDPAddr).Port
	c.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, byte(port >> 8), byte(port)})
	c.SetDeadline(time.Time{})
	session, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { io.Copy(io.Discard, c); cancel(); udp.Close() }()
	var mu sync.Mutex
	flows := map[string]net.Conn{}
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		for _, f := range flows {
			f.Close()
		}
	}()
	buf := make([]byte, 65535)
	var owner *net.UDPAddr
	for {
		udp.SetReadDeadline(time.Now().Add(time.Second))
		n, from, e := udp.ReadFromUDP(buf)
		if e != nil {
			if session.Err() != nil {
				return nil
			}
			if t, ok := e.(net.Error); ok && t.Timeout() {
				continue
			}
			return e
		}
		if !from.IP.IsLoopback() || n < 4 || buf[0] != 0 || buf[1] != 0 || buf[2] != 0 {
			continue
		}
		if owner != nil && owner.String() != from.String() {
			continue
		}
		if owner == nil {
			owner = from
		}
		reader := &sliceReader{data: buf[3:n]}
		address, raw, e := readSOCKSAddress(reader)
		if e != nil {
			continue
		}
		payload := append([]byte(nil), reader.data...)
		mu.Lock()
		remote := flows[address]
		mu.Unlock()
		if remote == nil {
			dialCtx, stop := context.WithTimeout(session, 10*time.Second)
			remote, e = stack.DialContext(dialCtx, "udp", address)
			stop()
			if e != nil {
				continue
			}
			mu.Lock()
			flows[address] = remote
			mu.Unlock()
			go func(r net.Conn, header []byte, target *net.UDPAddr) {
				defer r.Close()
				defer func() { mu.Lock(); delete(flows, address); mu.Unlock() }()
				reply := make([]byte, 65535)
				for {
					r.SetReadDeadline(time.Now().Add(90 * time.Second))
					n, e := r.Read(reply)
					if e != nil {
						return
					}
					packet := append([]byte{0, 0, 0}, header...)
					packet = append(packet, reply[:n]...)
					if _, e = udp.WriteToUDP(packet, target); e != nil {
						return
					}
				}
			}(remote, raw, from)
		}
		remote.Write(payload)
	}
}

type sliceReader struct{ data []byte }

func (r *sliceReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}
