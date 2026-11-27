package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/core"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const chromeOrigin = "chrome-extension://gmhibjohmhkchkongeibochdfidjiiic/"

type chromeRequest struct {
	ID           int    `json:"id"`
	Action       string `json:"action"`
	ProbeURL     string `json:"probeURL"`
	Link         string `json:"link"`
	Name         string `json:"name"`
	NodeID       string `json:"nodeId"`
	Confirmed    bool   `json:"confirmed"`
	HandoffID    int    `json:"handoffId"`
	HandoffError string `json:"handoffError"`
}
type chromeNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}
type chromeResponse struct {
	ID             int          `json:"id"`
	OK             bool         `json:"ok"`
	Error          string       `json:"error,omitempty"`
	Connected      bool         `json:"connected"`
	ActiveNodeID   string       `json:"activeNodeId"`
	Nodes          []chromeNode `json:"nodes"`
	Port           int          `json:"port"`
	Event          string       `json:"event,omitempty"`
	DesktopRunning bool         `json:"desktopRunning"`
	HandoffID      int          `json:"handoffId,omitempty"`
}

func readNativeMessage(r io.Reader, target any) error {
	var length uint32
	if e := binary.Read(r, binary.LittleEndian, &length); e != nil {
		return e
	}
	if length == 0 || length > 1024*1024 {
		return fmt.Errorf("invalid native message length")
	}
	b := make([]byte, length)
	if _, e := io.ReadFull(r, b); e != nil {
		return e
	}
	return json.Unmarshal(b, target)
}
func writeNativeMessage(w io.Writer, value any) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	if e = binary.Write(w, binary.LittleEndian, uint32(len(b))); e != nil {
		return e
	}
	_, e = w.Write(b)
	return e
}
func browserSnapshot(b *core.Backend) chromeResponse {
	cfg := b.GetConfig()
	state := b.GetState()
	nodes := []chromeNode{}
	for _, n := range cfg.Nodes {
		p := n.Protocol
		if p == "" {
			p = "sudoku"
		}
		nodes = append(nodes, chromeNode{n.ID, n.Name, p, n.ServerAddress})
	}
	return chromeResponse{OK: true, Connected: state.Running && state.CoreRunning, ActiveNodeID: cfg.ActiveNodeID, Nodes: nodes, Port: 17991}
}
func runChromeHost() bool {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "chrome-extension://") {
		return false
	}
	if os.Args[1] != chromeOrigin {
		return true
	}
	unlock, e := lockChromeHelper()
	if e != nil {
		writeNativeMessage(os.Stdout, chromeResponse{Error: e.Error()})
		return true
	}
	defer unlock()
	// Chrome may terminate its native host without allowing deferred shutdown.
	// Keep every browser core tied to the host process lifetime.
	releaseProcesses, e := containChromeProcesses()
	if e != nil {
		writeNativeMessage(os.Stdout, chromeResponse{Error: "Не удалось защитить жизненный цикл помощника Chrome"})
		return true
	}
	defer releaseProcesses()
	b, e := core.NewBrowserBackend(bundledRuntime, "runtime/bin")
	if e != nil {
		writeNativeMessage(os.Stdout, chromeResponse{Error: "Не удалось загрузить профили расширения"})
		return true
	}
	b.Startup(context.Background())
	defer b.Shutdown()
	done := make(chan struct{})
	defer close(done)
	var output sync.Mutex
	send := func(v chromeResponse) { output.Lock(); defer output.Unlock(); writeNativeMessage(os.Stdout, v) }
	var ackMu sync.Mutex
	var handoffID int
	var ack chan error
	listener, e := serveVPNPeer("browser", func(command string) (vpnPeerState, error) {
		state := browserSnapshot(b)
		switch command {
		case "status":
			return vpnPeerState{Running: state.Connected}, nil
		case "release":
			desktop, err := contactVPNPeer("desktop", "status", 2*time.Second)
			if err != nil || !desktop.Running {
				return vpnPeerState{}, errors.New("Системный VPN ещё не подключён")
			}
			ackMu.Lock()
			handoffID++
			id := handoffID
			reply := make(chan error, 1)
			ack = reply
			ackMu.Unlock()
			defer func() {
				ackMu.Lock()
				if handoffID == id {
					ack = nil
				}
				ackMu.Unlock()
			}()
			state.Event = "release-browser"
			state.HandoffID = id
			send(state)
			select {
			case err := <-reply:
				if err != nil {
					return vpnPeerState{}, err
				}
			case <-time.After(12 * time.Second):
				return vpnPeerState{}, errors.New("Chrome не подтвердил переключение")
			case <-done:
				return vpnPeerState{}, errors.New("Chrome закрыт")
			}
			// Chrome has already switched to the verified desktop route.
			if err := b.StopProxy(); err != nil {
				return vpnPeerState{}, fmt.Errorf("Не удалось остановить браузерное ядро: %w", err)
			}
			return vpnPeerState{}, nil
		default:
			return vpnPeerState{}, errors.New("unknown browser handoff command")
		}
	})
	if e != nil {
		send(chromeResponse{Error: e.Error()})
		return true
	}
	defer listener.Close()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		previous := false
		previousDesktop := false
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				state := browserSnapshot(b)
				if previous && !state.Connected {
					state.Event = "stopped"
					state.DesktopRunning = previousDesktop
					send(state)
				}
				previous = state.Connected
				desktop, e := contactVPNPeer("desktop", "status", time.Second)
				if e == nil && previousDesktop != desktop.Running {
					previousDesktop = desktop.Running
					state.DesktopRunning = desktop.Running
					state.Event = "desktop-status"
					send(state)
				}
			}
		}
	}()
	for {
		var req chromeRequest
		if e := readNativeMessage(os.Stdin, &req); e != nil {
			return true
		}
		if req.Action == "handoff-ack" {
			ackMu.Lock()
			if req.HandoffID == handoffID && ack != nil {
				var err error
				if req.HandoffError != "" {
					err = errors.New(req.HandoffError)
				}
				select {
				case ack <- err:
				default:
				}
			}
			ackMu.Unlock()
			continue
		}
		var err error
		switch req.Action {
		case "status":
		case "import":
			_, err = b.ImportShortLink(req.Name, req.Link)
		case "select":
			err = b.SwitchNode(req.NodeID)
		case "delete":
			if b.GetConfig().ActiveNodeID == req.NodeID {
				err = b.StopProxy()
			}
			if err == nil {
				err = b.DeleteNode(req.NodeID)
			}
		case "connect":
			err = connectBrowserVPN(b, req)
		case "disconnect":
			err = b.StopProxy()
		default:
			err = fmt.Errorf("неизвестная команда")
		}
		response := browserSnapshot(b)
		response.ID = req.ID
		desktop, statusErr := contactVPNPeer("desktop", "status", time.Second)
		response.DesktopRunning = desktop.Running
		if req.Action == "status" && err == nil {
			err = statusErr
		}
		if err != nil {
			response.OK = false
			response.Error = err.Error()
		}
		send(response)
	}
}

func startBrowserProxy(b *core.Backend, probeURL string) error {
	err := b.StartProxy(core.StartRequest{WithTun: false})
	if err == nil {
		deadline := time.Now().Add(25 * time.Second)
		for {
			conn, e := net.DialTimeout("tcp", "127.0.0.1:17991", 300*time.Millisecond)
			if e == nil {
				conn.SetDeadline(time.Now().Add(time.Second))
				conn.Write([]byte{5, 1, 0})
				var reply [2]byte
				_, e = io.ReadFull(conn, reply[:])
				conn.Close()
				if e == nil && reply == [2]byte{5, 0} {
					break
				}
			}
			if time.Now().After(deadline) || !b.GetState().CoreRunning {
				err = fmt.Errorf("Не удалось запустить соединение. Проверьте профиль сервера")
				b.StopProxy()
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err == nil {
		err = b.CheckProxyURL(probeURL)
		if err != nil {
			_ = b.StopProxy()
		}
	}
	return err
}

func connectBrowserVPN(b *core.Backend, req chromeRequest) error {
	release, err := lockVPNTransition()
	if err != nil {
		return err
	}
	defer release()
	desktop, err := contactVPNPeer("desktop", "status", 2*time.Second)
	if err != nil {
		return err
	}
	if desktop.Running && !req.Confirmed {
		return errDesktopVPNActive
	}
	if err := startBrowserProxy(b, req.ProbeURL); err != nil {
		return err
	}
	if !desktop.Running {
		return nil
	}
	if _, err = contactVPNPeer("desktop", "release", 35*time.Second); err == nil {
		err = b.CheckProxyURL(req.ProbeURL)
	}
	if err != nil {
		_, restoreErr := contactVPNPeer("desktop", "restore", 50*time.Second)
		return fmt.Errorf("Переключение на VPN Chrome: %w", errors.Join(err, b.StopProxy(), restoreErr))
	}
	_, _ = contactVPNPeer("desktop", "commit", 2*time.Second)
	return nil
}
