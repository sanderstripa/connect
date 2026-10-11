package core

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Replace only the SOCKS engine. HEV's MapDNS cache, adapter, system DNS and
// default routes remain intact. Existing streams close; new streams use the new core.
func (b *Backend) switchCoreKeepingTun(previous NodeConfig) (bool, error) {
	if runtime.GOOS != "windows" || b.browserOnly {
		return false, nil
	}
	b.opMu.Lock()
	defer b.opMu.Unlock()
	b.mu.Lock()
	route := b.routeState
	next := b.findNode(b.cfg.ActiveNodeID)
	if route == nil || next == nil || !b.tunRunningLocked() || b.dnsProxy != nil ||
		protocolOf(previous) == "mist" || protocolOf(*next) == "mist" {
		b.mu.Unlock()
		return false, nil
	}
	// Hostname endpoints retain the full setup path until their complete address
	// set can be pinned through the physical DNS resolver.
	host, _, err := net.SplitHostPort(next.ServerAddress)
	if err != nil || net.ParseIP(host).To4() == nil {
		b.mu.Unlock()
		return false, nil
	}
	plan, err := b.prepareProxyStartLocked(StartRequest{WithTun: true})
	if err != nil {
		b.mu.Unlock()
		_ = b.SetActiveNode(previous.ID)
		return true, err
	}
	if plan.localPort != b.runningLocalPort {
		b.mu.Unlock()
		return false, nil
	}
	oldIP := route.ServerIP
	newIP := net.ParseIP(host).To4().String()
	ctx, startID := b.newStartContext()
	defer b.clearStartIfMatch(startID)
	b.coreSwitching = true
	b.state.Running = false
	b.state.CoreRunning = false
	b.emitStateLocked()
	b.mu.Unlock()
	defer func() { b.mu.Lock(); b.coreSwitching = false; b.emitStateLocked(); b.mu.Unlock() }()
	started := time.Now()
	if newIP != oldIP {
		if err = windowsSwitchHostRoute(ctx, route, newIP, true); err != nil {
			_ = b.SetActiveNode(previous.ID)
			b.mu.Lock()
			b.state.Running = b.coreProc.IsRunning()
			b.state.CoreRunning = b.state.Running
			b.mu.Unlock()
			return true, err
		}
	}
	err = b.coreProc.Stop(2 * time.Second)
	if err == nil {
		err = b.startSwitchedCore(ctx, plan, route.WindowsDefaultIfIndex)
	}
	if err != nil {
		_ = b.coreProc.Stop(2 * time.Second)
		_ = b.SetActiveNode(previous.ID)
		b.mu.Lock()
		restore, prepareErr := b.prepareProxyStartLocked(StartRequest{WithTun: true})
		b.mu.Unlock()
		if prepareErr == nil && ctx.Err() == nil {
			prepareErr = b.startSwitchedCore(ctx, restore, route.WindowsDefaultIfIndex)
		}
		if newIP != oldIP {
			_ = windowsSwitchHostRoute(context.Background(), route, newIP, false)
		}
		b.mu.Lock()
		b.state.LastError = err.Error()
		b.mu.Unlock()
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if prepareErr != nil {
			return true, fmt.Errorf("переключение: %w; восстановление: %v", err, prepareErr)
		}
		return true, fmt.Errorf("переключение: %w; предыдущее соединение сохранено", err)
	}
	b.mu.Lock()
	route.ServerIP = newIP
	route.WindowsFirewallRule = ""
	checkpointErr := b.saveWindowsRouteCheckpoint(route, plan.tun)
	b.mu.Unlock()
	if newIP != oldIP {
		_ = windowsSwitchHostRoute(context.Background(), route, oldIP, false)
	}
	if checkpointErr != nil {
		b.addLog("warn", "route", checkpointErr.Error())
	}
	b.addLog("info", "app", fmt.Sprintf("switched to %s in %s; TUN and DNS retained", protocolOf(plan.node), time.Since(started).Round(time.Millisecond)))
	return true, nil
}

func (b *Backend) startSwitchedCore(ctx context.Context, plan proxyStartPlan, ifIndex int) error {
	cache := filepath.Join(b.store.RuntimeDir(), "cache", "sudoku-rules")
	env := baseCoreEnvironment(plan.coreLogLevel, plan.trafficStatsFile, cache)
	env = append(env, fmt.Sprintf("SUDOKU_OUTBOUND_IFINDEX=%d", ifIndex), fmt.Sprintf("CONNECT_OUTBOUND_IFINDEX=%d", ifIndex))
	if err := b.coreProc.Start(plan.sudokuBinary, engineArgs(plan.node, plan.sudokuConfigPath, plan.localPort), env, plan.workDir, b.onCoreLog); err != nil {
		return err
	}
	if err := b.preflightTunUpstream(ctx, plan.localPort, plan.routing, time.Now()); err != nil {
		return err
	}
	probeURL := ""
	if b.preflightTarget != "" {
		probeURL = "http://" + b.preflightTarget
	}
	if err := checkProxyURL(ctx, probeURL, plan.localPort, 5*time.Second); err != nil {
		return err
	}
	b.mu.Lock()
	b.state.Running = true
	b.state.CoreRunning = true
	b.state.ActiveNodeID = plan.node.ID
	b.state.ActiveNodeName = plan.node.Name
	b.state.StartedAtUnix = time.Now().UnixMilli()
	b.state.LastError = ""
	b.trafficCache = trafficSampleState{coreTrafficFile: plan.trafficStatsFile}
	b.resetKernelLatencyLocked()
	b.cfg.LastStartedNode = plan.node.ID
	_ = b.store.Save(b.cfg)
	b.mu.Unlock()
	return nil
}

func windowsSwitchHostRoute(ctx context.Context, route *routeContext, ip string, add bool) error {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.IsLoopback() {
		return nil
	}
	verb := "DELETE"
	if add {
		verb = "ADD"
	}
	args := []string{verb, ip, "MASK", "255.255.255.255", route.DefaultGateway, "IF", fmt.Sprint(route.WindowsDefaultIfIndex)}
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "route.exe", args...)
	applyManagedProcessSysProcAttr(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("маршрут сервера: %w: %s", err, out)
	}
	return nil
}
