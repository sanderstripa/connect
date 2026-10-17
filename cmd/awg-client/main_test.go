package main

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"golang.org/x/net/dns/dnsmessage"
	"io"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/awgconfig"
	"github.com/amnezia-vpn/amneziawg-go/v3/conn"
	"github.com/amnezia-vpn/amneziawg-go/v3/device"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun/netstack"
)

func TestEncryptedAWGSOCKSTCPUDP(t *testing.T) {
	for _, generation := range []string{"1", "2", "3"} {
		t.Run(generation, func(t *testing.T) {
			clientKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
			serverKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
			b64 := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
			fields := map[string]string{"Jc": "4", "Jmin": "40", "Jmax": "80", "S1": "32", "S2": "48", "H1": "123456", "H2": "234567", "H3": "345678", "H4": "456789"}
			if generation != "1" {
				fields["S3"] = "16"
				fields["S4"] = "24"
				fields["I1"] = "<b 0x1234><r 16>"
				fields["H1"] = "123456-123460"
			}
			if generation == "3" {
				k := make([]byte, 32)
				rand.Read(k)
				fields["HeaderProtectionKey"] = b64(k)
				fields["ContentPaddingAddition"] = "0-8"
			}
			var settings strings.Builder
			for k, v := range fields {
				settings.WriteString(k + " = " + v + "\n")
			}
			serverRaw := "[Interface]\nAddress = 10.64.0.1/24\nPrivateKey = " + b64(serverKey.Bytes()) + "\n" + settings.String() + "[Peer]\nPublicKey = " + b64(clientKey.PublicKey().Bytes()) + "\nEndpoint = 127.0.0.1:9\nAllowedIPs = 10.64.0.2/32\n"
			sc, e := awgconfig.Parse(serverRaw)
			if e != nil {
				t.Fatal(e)
			}
			tun, serverNet, e := netstack.CreateNetTUN([]netip.Addr{netip.MustParseAddr("10.64.0.1")}, nil, 1280)
			if e != nil {
				t.Fatal(e)
			}
			dev := device.NewDevice(tun, conn.NewStdNetBind(), device.NewLogger(device.LogLevelSilent, ""))
			defer dev.Close()
			uapi, e := sc.UAPI("127.0.0.1:9")
			if e != nil {
				t.Fatal(e)
			}
			if e = dev.IpcSet(uapi); e != nil {
				t.Fatal(e)
			}
			if e = dev.Up(); e != nil {
				t.Fatal(e)
			}
			info, e := dev.IpcGet()
			if e != nil {
				t.Fatal(e)
			}
			port := ""
			for _, line := range strings.Split(info, "\n") {
				if strings.HasPrefix(line, "listen_port=") {
					port = strings.TrimPrefix(line, "listen_port=")
				}
			}
			raw := "[Interface]\nAddress = 10.64.0.2/24\nPrivateKey = " + b64(clientKey.Bytes()) + "\n" + settings.String() + "[Peer]\nPublicKey = " + b64(serverKey.PublicKey().Bytes()) + "\nEndpoint = 127.0.0.1:" + port + "\nAllowedIPs = 0.0.0.0/0\n"
			cfg, e := awgconfig.Parse(raw)
			if e != nil {
				t.Fatal(e)
			}
			cfg.DNS = []string{"10.64.0.1"}
			resolver, e := serverNet.ListenUDPAddrPort(netip.MustParseAddrPort("10.64.0.1:53"))
			if e != nil {
				t.Fatal(e)
			}
			defer resolver.Close()
			go func() {
				b := make([]byte, 2048)
				for {
					n, addr, e := resolver.ReadFrom(b)
					if e != nil {
						return
					}
					var q dnsmessage.Message
					if q.Unpack(b[:n]) != nil {
						continue
					}
					answer := dnsmessage.Message{Header: dnsmessage.Header{ID: q.ID, Response: true, RecursionAvailable: true}, Questions: q.Questions}
					for _, question := range q.Questions {
						if question.Type == dnsmessage.TypeA {
							answer.Answers = append(answer.Answers, dnsmessage.Resource{Header: dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 30}, Body: &dnsmessage.AResource{A: [4]byte{10, 64, 0, 1}}})
						}
					}
					packet, e := answer.Pack()
					if e == nil {
						resolver.WriteTo(packet, addr)
					}
				}
			}()
			tcp, e := serverNet.ListenTCPAddrPort(netip.MustParseAddrPort("10.64.0.1:8080"))
			if e != nil {
				t.Fatal(e)
			}
			defer tcp.Close()
			go func() {
				for {
					c, e := tcp.Accept()
					if e != nil {
						return
					}
					go func() { defer c.Close(); io.Copy(c, c) }()
				}
			}()
			udp, e := serverNet.ListenUDPAddrPort(netip.MustParseAddrPort("10.64.0.1:8081"))
			if e != nil {
				t.Fatal(e)
			}
			defer udp.Close()
			go func() {
				b := make([]byte, 4096)
				for {
					n, a, e := udp.ReadFrom(b)
					if e != nil {
						return
					}
					udp.WriteTo(b[:n], a)
				}
			}()
			reserve, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			listen := reserve.Addr().String()
			reserve.Close()
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- run(ctx, cfg, listen) }()
			defer func() {
				cancel()
				select {
				case e := <-done:
					if e != nil {
						t.Error(e)
					}
				case <-time.After(5 * time.Second):
					t.Error("core did not stop")
				}
			}()
			var c net.Conn
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				c, e = net.DialTimeout("tcp", listen, time.Second)
				if e == nil {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if e != nil {
				t.Fatal(e)
			}
			defer c.Close()
			handshake := func(c net.Conn, request []byte) []byte {
				t.Helper()
				c.SetDeadline(time.Now().Add(15 * time.Second))
				c.Write([]byte{5, 1, 0})
				b := make([]byte, 2)
				if _, e := io.ReadFull(c, b); e != nil || b[1] != 0 {
					t.Fatalf("greeting %v %v", b, e)
				}
				c.Write(request)
				b = make([]byte, 10)
				if _, e := io.ReadFull(c, b); e != nil || b[1] != 0 {
					t.Fatalf("request %v %v", b, e)
				}
				return b
			}
			handshake(c, []byte{5, 1, 0, 1, 10, 64, 0, 1, 31, 144})
			for i := 0; i < 3; i++ {
				c.Write([]byte("awg-tcp"))
				b := make([]byte, 7)
				if _, e = io.ReadFull(c, b); e != nil || string(b) != "awg-tcp" {
					t.Fatal("encrypted TCP failed", e)
				}
			}
			domainConn, e := net.Dial("tcp", listen)
			if e != nil {
				t.Fatal(e)
			}
			defer domainConn.Close()
			handshake(domainConn, append(append([]byte{5, 1, 0, 3, 8}, []byte("awg.test")...), 31, 144))
			domainConn.Write([]byte("dns"))
			domainReply := make([]byte, 3)
			if _, e = io.ReadFull(domainConn, domainReply); e != nil || string(domainReply) != "dns" {
				t.Fatal("tunneled DNS failed", e)
			}
			control, e := net.Dial("tcp", listen)
			if e != nil {
				t.Fatal(e)
			}
			defer control.Close()
			reply := handshake(control, []byte{5, 3, 0, 1, 0, 0, 0, 0, 0, 0})
			sock, e := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(binary.BigEndian.Uint16(reply[8:]))})
			if e != nil {
				t.Fatal(e)
			}
			defer sock.Close()
			sock.SetDeadline(time.Now().Add(5 * time.Second))
			packet := append([]byte{0, 0, 0, 1, 10, 64, 0, 1, 31, 145}, []byte("awg-udp")...)
			sock.Write(packet)
			b := make([]byte, 128)
			n, e := sock.Read(b)
			if e != nil || n < 10 || string(b[10:n]) != "awg-udp" {
				t.Fatal("encrypted UDP failed", e)
			}
		})
	}
}
