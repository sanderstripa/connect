package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/awgconfig"
	"github.com/amnezia-vpn/amneziawg-go/v3/conn"
	"github.com/amnezia-vpn/amneziawg-go/v3/device"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun/netstack"
)

const buildVersion = "connect-0.1.8+awg-3.1.20260828"

func main() {
	file := flag.String("config", "", "Connect AmneziaWG JSON configuration")
	listen := flag.String("socks", "127.0.0.1:1080", "loopback SOCKS endpoint")
	version := flag.Bool("version", false, "print version")
	flag.Parse()
	if *version {
		fmt.Println(buildVersion)
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	raw, e := os.ReadFile(*file)
	if e != nil {
		log.Fatal("AmneziaWG: cannot read configuration")
	}
	var cfg awgconfig.Config
	if json.Unmarshal(raw, &cfg) != nil {
		log.Fatal("AmneziaWG: invalid configuration")
	}
	if e = run(ctx, cfg, *listen); e != nil && ctx.Err() == nil {
		log.Fatal(e)
	}
}

func run(ctx context.Context, cfg awgconfig.Config, listen string) error {
	if e := cfg.Validate(); e != nil {
		return e
	}
	host, _, e := net.SplitHostPort(listen)
	if e != nil || host != "127.0.0.1" {
		return fmt.Errorf("AmneziaWG: SOCKS must bind to 127.0.0.1")
	}
	endpointHost, port, _ := net.SplitHostPort(cfg.Peer["Endpoint"])
	// Resolve before Windows installs its TUN routes. Never resolve the VPN
	// endpoint through the tunnel itself.
	resolveCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	ips, e := net.DefaultResolver.LookupNetIP(resolveCtx, "ip", endpointHost)
	stop()
	if e != nil || len(ips) == 0 {
		return fmt.Errorf("AmneziaWG: cannot resolve server")
	}
	ip := ips[0].Unmap()
	for _, p := range ips {
		p = p.Unmap()
		if p.Is4() {
			ip = p
			break
		}
	}
	uapi, e := cfg.UAPI(net.JoinHostPort(ip.String(), port))
	if e != nil {
		return e
	}
	addresses := []netip.Addr{}
	dns := []netip.Addr{}
	for _, s := range cfg.Address {
		p, _ := netip.ParsePrefix(s)
		addresses = append(addresses, p.Addr())
	}
	for _, s := range cfg.DNS {
		dns = append(dns, netip.MustParseAddr(s))
	}
	// A resolver is required for Chrome's domain requests. It is reached only
	// through AWG and is subject to AllowedIPs, with no direct DNS fallback.
	if len(dns) == 0 {
		dns = append(dns, netip.MustParseAddr("1.1.1.1"))
	}
	tun, stack, e := netstack.CreateNetTUN(addresses, dns, cfg.MTU)
	if e != nil {
		return fmt.Errorf("AmneziaWG: cannot create userspace network")
	}
	bind := conn.NewStdNetBind()
	dev := device.NewDevice(tun, bind, device.NewLogger(device.LogLevelSilent, ""))
	defer dev.Close()
	if dev.IpcSet(uapi) != nil {
		return fmt.Errorf("AmneziaWG: core rejected configuration parameters")
	}
	if e = dev.Up(); e != nil {
		return fmt.Errorf("AmneziaWG: cannot start transport")
	}
	if s := os.Getenv("CONNECT_OUTBOUND_IFINDEX"); s != "" {
		index, err := strconv.ParseUint(s, 10, 32)
		if err != nil || index == 0 {
			return fmt.Errorf("AmneziaWG: invalid physical interface")
		}
		if b, ok := bind.(conn.BindSocketToInterface); ok {
			if ip.Is4() {
				e = b.BindSocketToInterface4(uint32(index), false)
			} else {
				e = b.BindSocketToInterface6(uint32(index), false)
			}
			if e != nil {
				return fmt.Errorf("AmneziaWG: cannot bind physical interface")
			}
		}
	}
	listener, e := net.Listen("tcp", listen)
	if e != nil {
		return e
	}
	defer listener.Close()
	session, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { <-session.Done(); listener.Close() }()
	log.Printf("CONNECT_READY AmneziaWG SOCKS %s", listen)
	for {
		c, err := listener.Accept()
		if err != nil {
			if session.Err() != nil {
				return nil
			}
			return err
		}
		go serveConnectSOCKS(session, c, stack)
	}
}
