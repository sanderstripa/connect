package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App struct
type App struct {
	backend      *core.Backend
	remote       *networkClient
	shutdownOnce sync.Once
	ready        chan struct{}

	runtimeFS   fs.FS
	runtimeRoot string
}

// NewApp creates a new App application struct
func NewApp(runtimeFS fs.FS, runtimeRoot string) *App {
	if needsNetworkHelper() {
		return &App{runtimeFS: runtimeFS, runtimeRoot: runtimeRoot, ready: make(chan struct{})}
	}
	backend, _ := core.NewBackendWithRuntimeFS(runtimeFS, runtimeRoot)
	return &App{
		backend:     backend,
		ready:       make(chan struct{}),
		runtimeFS:   runtimeFS,
		runtimeRoot: runtimeRoot,
	}
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	defer close(a.ready)
	if needsNetworkHelper() {
		var err error
		a.remote, err = startNetworkHelper()
		return err
	}
	if a.backend == nil {
		b, err := core.NewBackendWithRuntimeFS(a.runtimeFS, a.runtimeRoot)
		if err != nil {
			return err
		}
		a.backend = b
	}
	if a.backend != nil {
		app := application.Get()
		a.backend.SetEventEmitter(func(name string, payload any) {
			if app != nil {
				app.Event.Emit(name, payload)
			}
		})
		a.backend.Startup(ctx)
	}
	return nil
}

func (a *App) ServiceShutdown() error {
	a.shutdownBackend()
	return nil
}

func (a *App) shutdownBackend() {
	a.shutdownOnce.Do(func() {
		if a.remote != nil {
			_ = a.remote.call("Shutdown", nil, nil)
			_ = a.remote.client.Close()
			return
		}
		if a.backend != nil {
			a.backend.Shutdown()
		}
	})
}

func (a *App) ShutdownNow() {
	a.shutdownBackend()
}

func (a *App) trayIsRunning() bool {
	state := a.GetState()
	return state.Running
}

func (a *App) trayToggleProxy() error {
	if a.trayIsRunning() {
		return a.StopProxy()
	}
	cfg := a.GetConfig()
	return a.StartProxy(core.StartRequest{WithTun: cfg.Tun.Enabled})
}

func (a *App) traySetProxyMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "global", "direct", "pac":
	default:
		return nil
	}
	return a.SetRoutingMode(mode)
}

func (a *App) trayCurrentProxyMode() string {
	cfg := a.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.Routing.ProxyMode))
	switch mode {
	case "global", "direct", "pac":
		return mode
	default:
		return "pac"
	}
}

func (a *App) GetConfig() core.AppConfig {
	if a.remote != nil {
		var result core.AppConfig
		err := a.remote.call("GetConfig", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return core.AppConfig{}
	}
	return a.backend.GetConfig()
}

func (a *App) SaveConfig(cfg core.AppConfig) error {
	if a.remote != nil {
		return a.remote.call("SaveConfig", []any{cfg}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.SaveConfig(cfg)
}

func (a *App) SetRoutingMode(mode string) error {
	if a.remote != nil {
		return a.remote.call("SetRoutingMode", []any{mode}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.SetRoutingMode(mode)
}

func (a *App) GetState() core.RuntimeState {
	if a.remote != nil {
		var result core.RuntimeState
		err := a.remote.call("GetState", []any{}, &result)
		if err != nil {
			result.LastError = err.Error()
		}
		return result
	}

	if a.backend == nil {
		return core.RuntimeState{}
	}
	return a.backend.GetState()
}

func (a *App) StartProxy(req core.StartRequest) error {
	if a.remote != nil {
		return a.remote.call("StartProxy", []any{req}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.StartProxy(req)
}

func (a *App) StopProxy() error {
	if a.remote != nil {
		return a.remote.call("StopProxy", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.StopProxy()
}

func (a *App) RestartProxy(req core.StartRequest) error {
	if a.remote != nil {
		return a.remote.call("RestartProxy", []any{req}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.RestartProxy(req)
}

func (a *App) UpsertNode(node core.NodeConfig) (core.NodeConfig, error) {
	if a.remote != nil {
		var result core.NodeConfig
		err := a.remote.call("UpsertNode", []any{node}, &result)
		return result, err
	}

	if a.backend == nil {
		return core.NodeConfig{}, nil
	}
	return a.backend.UpsertNode(node)
}

func (a *App) DeleteNode(nodeID string) error {
	if a.remote != nil {
		return a.remote.call("DeleteNode", []any{nodeID}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.DeleteNode(nodeID)
}

func (a *App) SetActiveNode(nodeID string) error {
	if a.remote != nil {
		return a.remote.call("SetActiveNode", []any{nodeID}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.SetActiveNode(nodeID)
}

func (a *App) SwitchNode(nodeID string) error {
	if a.remote != nil {
		return a.remote.call("SwitchNode", []any{nodeID}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.SwitchNode(nodeID)
}

func (a *App) ImportShortLink(name string, link string) (core.NodeConfig, error) {
	if a.remote != nil {
		var result core.NodeConfig
		err := a.remote.call("ImportShortLink", []any{name, link}, &result)
		return result, err
	}

	if a.backend == nil {
		return core.NodeConfig{}, nil
	}
	return a.backend.ImportShortLink(name, link)
}

func (a *App) ExportShortLink(nodeID string) (string, error) {
	if a.remote != nil {
		var result string
		err := a.remote.call("ExportShortLink", []any{nodeID}, &result)
		return result, err
	}

	if a.backend == nil {
		return "", nil
	}
	return a.backend.ExportShortLink(nodeID)
}

func (a *App) ProbeNode(nodeID string) (core.LatencyResult, error) {
	if a.remote != nil {
		var result core.LatencyResult
		err := a.remote.call("ProbeNode", []any{nodeID}, &result)
		return result, err
	}

	if a.backend == nil {
		return core.LatencyResult{}, nil
	}
	return a.backend.ProbeNode(nodeID)
}

func (a *App) ProbeAllNodes() []core.LatencyResult {
	if a.remote != nil {
		var result []core.LatencyResult
		err := a.remote.call("ProbeAllNodes", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.ProbeAllNodes()
}

func (a *App) AutoSelectLowestLatency() (core.LatencyResult, error) {
	if a.remote != nil {
		var result core.LatencyResult
		err := a.remote.call("AutoSelectLowestLatency", []any{}, &result)
		return result, err
	}

	if a.backend == nil {
		return core.LatencyResult{}, nil
	}
	return a.backend.AutoSelectLowestLatency()
}

func (a *App) SortNodesByName() error {
	if a.remote != nil {
		return a.remote.call("SortNodesByName", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.SortNodesByName()
}

func (a *App) SortNodesByLatency() error {
	if a.remote != nil {
		return a.remote.call("SortNodesByLatency", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.SortNodesByLatency()
}

func (a *App) DetectIPDirect() core.IPDetectResult {
	if a.remote != nil {
		var result core.IPDetectResult
		err := a.remote.call("DetectIPDirect", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return core.IPDetectResult{}
	}
	return a.backend.DetectIPDirect()
}

func (a *App) DetectIPProxy() core.IPDetectResult {
	if a.remote != nil {
		var result core.IPDetectResult
		err := a.remote.call("DetectIPProxy", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return core.IPDetectResult{}
	}
	return a.backend.DetectIPProxy()
}

func (a *App) GetLANProxyInfo() core.LANProxyInfo {
	if a.remote != nil {
		var result core.LANProxyInfo
		err := a.remote.call("GetLANProxyInfo", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return core.LANProxyInfo{}
	}
	return a.backend.GetLANProxyInfo()
}

func (a *App) StartReverseForwarder() error {
	if a.remote != nil {
		return a.remote.call("StartReverseForwarder", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.StartReverseForwarder()
}

func (a *App) StopReverseForwarder() error {
	if a.remote != nil {
		return a.remote.call("StopReverseForwarder", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.StopReverseForwarder()
}

func (a *App) GetLogs(level string, limit int) []core.LogEntry {
	if a.remote != nil {
		var result []core.LogEntry
		err := a.remote.call("GetLogs", []any{level, limit}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.GetLogs(level, limit)
}

func (a *App) ExportLogs(entries []core.LogEntry) (string, error) {
	if a.remote != nil {
		var result string
		err := a.remote.call("ExportLogs", []any{entries}, &result)
		return result, err
	}

	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("application is not ready")
	}
	filename := "sudoku-logs-" + time.Now().Format("2006-01-02-15-04-05") + ".csv"
	path, err := app.Dialog.SaveFile().
		SetFilename(filename).
		AddFilter("CSV Files", "*.csv").
		PromptForSingleSelection()
	if err != nil || strings.TrimSpace(path) == "" {
		return "", err
	}
	if filepath.Ext(path) == "" {
		path += ".csv"
	}
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	writer := csv.NewWriter(file)
	writeErr := writer.Write([]string{"timestamp", "level", "component", "message"})
	for _, entry := range entries {
		if writeErr != nil {
			break
		}
		writeErr = writer.Write([]string{
			entry.Timestamp.Format(time.RFC3339Nano),
			entry.Level,
			entry.Component,
			firstNonEmpty(entry.Message, entry.Raw),
		})
	}
	writer.Flush()
	if writeErr == nil {
		writeErr = writer.Error()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return path, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (a *App) GetConnections() []core.ActiveConnection {
	if a.remote != nil {
		var result []core.ActiveConnection
		err := a.remote.call("GetConnections", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.GetConnections()
}

func (a *App) CloseConnection(connectionID string) error {
	if a.remote != nil {
		return a.remote.call("CloseConnection", []any{connectionID}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.CloseConnection(connectionID)
}

func (a *App) CloseAllConnections() error {
	if a.remote != nil {
		return a.remote.call("CloseAllConnections", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.CloseAllConnections()
}

func (a *App) GetUsageHistory(limit int) []core.UsageDay {
	if a.remote != nil {
		var result []core.UsageDay
		err := a.remote.call("GetUsageHistory", []any{limit}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.GetUsageHistory(limit)
}

func (a *App) EnsureCoreBinaries() error {
	if a.remote != nil {
		return a.remote.call("EnsureCoreBinaries", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.EnsureCoreBinaries()
}

func (a *App) BuildInfo() map[string]string {
	if a.remote != nil {
		var result map[string]string
		err := a.remote.call("BuildInfo", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return map[string]string{}
	}
	return a.backend.BuildInfo()
}

func (a *App) ValidateYAML(content string) error {
	if a.remote != nil {
		return a.remote.call("ValidateYAML", []any{content}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.ValidateYAML(content)
}

func (a *App) OpenRuntimeDir() string {
	if a.remote != nil {
		var result string
		err := a.remote.call("OpenRuntimeDir", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return ""
	}
	return a.backend.OpenRuntimeDir()
}

func (a *App) OpenConfigPath() string {
	if a.remote != nil {
		var result string
		err := a.remote.call("OpenConfigPath", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return ""
	}
	return a.backend.OpenConfigPath()
}

// TunHasPrivileges reports whether the app can perform privileged TUN operations
// silently on the current platform (supported on macOS/Linux via in-app sudo caching).
func (a *App) TunHasPrivileges() bool {
	if a.remote != nil {
		var result bool
		err := a.remote.call("TunHasPrivileges", []any{}, &result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return result
	}

	if a.backend == nil {
		return false
	}
	return a.backend.TunHasPrivileges()
}

// TunAcquirePrivileges validates and caches the system password for sudo-backed TUN
// operations. The password is stored in-memory only and never written to disk.
func (a *App) TunAcquirePrivileges(password string) error {
	if a.remote != nil {
		return a.remote.call("TunAcquirePrivileges", []any{password}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.TunAcquirePrivileges(password)
}

// TunDropPrivileges clears any cached admin credentials used for TUN operations.
func (a *App) TunDropPrivileges() error {
	if a.remote != nil {
		return a.remote.call("TunDropPrivileges", []any{}, nil)
	}

	if a.backend == nil {
		return nil
	}
	return a.backend.TunDropPrivileges()
}
