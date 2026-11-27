package main

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/core"
)

type vpnPeerState struct {
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}

var errBrowserVPNActive = errors.New("CONNECT_BROWSER_VPN_ACTIVE")
var errDesktopVPNActive = errors.New("CONNECT_DESKTOP_VPN_ACTIVE")

func (a *App) BrowserVPNActive() (bool, error) {
	state, err := contactVPNPeer("browser", "status", 2*time.Second)
	return state.Running, err
}

func (a *App) StartProxyWithBrowserHandoff(req core.StartRequest) error {
	if a.remote != nil {
		return a.remote.call("StartProxyWithBrowserHandoff", []any{req}, nil)
	}
	return a.startDesktopVPN(req, true)
}

func (a *App) startDesktopVPN(req core.StartRequest, confirmed bool) error {
	if a.backend == nil {
		return nil
	}
	release, err := lockVPNTransition()
	if err != nil {
		return err
	}
	defer release()
	browser, err := a.BrowserVPNActive()
	if err != nil {
		return err
	}
	if browser && !confirmed {
		return errBrowserVPNActive
	}
	wasRunning := a.backend.GetState().Running
	if err := a.backend.StartProxy(req); err != nil {
		return err
	}
	if !browser {
		return nil
	}
	state := a.backend.GetState()
	if !state.Running || !state.CoreRunning || (req.WithTun && !state.TunRunning) {
		err := errors.New("Защищённое системное подключение не готово")
		if !wasRunning {
			err = errors.Join(err, a.backend.StopProxy())
		}
		return err
	}
	if _, err := contactVPNPeer("browser", "release", 18*time.Second); err != nil {
		// Chrome may already have released its proxy even if the reply was lost.
		// Keep the verified desktop route in place to protect its traffic.
		return fmt.Errorf("Системный VPN включён, Chrome не подтвердил переключение: %w", err)
	}
	return nil
}

func (a *App) serveDesktopVPN() (io.Closer, error) {
	var mu sync.Mutex
	var rollbackUntil time.Time
	var previous core.StartRequest
	return serveVPNPeer("desktop", func(command string) (vpnPeerState, error) {
		state := a.backend.GetState()
		if command == "status" {
			return vpnPeerState{Running: state.Running || state.TunRunning}, nil
		}
		mu.Lock()
		defer mu.Unlock()
		switch command {
		case "release":
			browser, err := contactVPNPeer("browser", "status", 2*time.Second)
			if err != nil || !browser.Running {
				return vpnPeerState{}, errors.New("VPN Chrome ещё не подключён")
			}
			rollbackUntil = time.Time{}
			state = a.backend.GetState()
			if !state.Running && !state.TunRunning {
				return vpnPeerState{}, nil
			}
			previous = core.StartRequest{WithTun: state.TunRunning}
			if err := a.backend.StopProxy(); err != nil {
				return vpnPeerState{}, err
			}
			rollbackUntil = time.Now().Add(25 * time.Second)
		case "restore":
			if time.Now().After(rollbackUntil) {
				if state := a.backend.GetState(); state.Running || state.TunRunning {
					return vpnPeerState{Running: true}, nil
				}
				return vpnPeerState{}, errors.New("Восстановление системного VPN больше недоступно")
			}
			rollbackUntil = time.Time{}
			if err := a.backend.StartProxy(previous); err != nil {
				return vpnPeerState{}, err
			}
		case "commit":
			rollbackUntil = time.Time{}
		default:
			return vpnPeerState{}, errors.New("unknown desktop handoff command")
		}
		state = a.backend.GetState()
		running := state.Running || state.TunRunning
		if command == "release" && running {
			return vpnPeerState{}, errors.New("Системный VPN по-прежнему активен")
		}
		return vpnPeerState{Running: running}, nil
	})
}
