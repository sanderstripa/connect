//go:build windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func vpnPipe(peer string) (string, string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", "", err
	}
	sid := user.User.Sid.String()
	return `\\.\pipe\connect-vpn-` + sid + "-" + peer, sid, nil
}

func serveVPNPeer(peer string, handle func(string) (vpnPeerState, error)) (io.Closer, error) {
	name, sid, err := vpnPipe(peer)
	if err != nil {
		return nil, err
	}
	listener, err := winio.ListenPipe(name, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;" + sid + ")(A;;GA;;;SY)S:(ML;;NW;;;ME)",
	})
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serveVPNCommand(conn, handle)
		}
	}()
	return listener, nil
}

func serveVPNCommand(conn net.Conn, handle func(string) (vpnPeerState, error)) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var command string
	if err := json.NewDecoder(io.LimitReader(conn, 128)).Decode(&command); err != nil {
		return
	}
	state, err := handle(command)
	if err != nil {
		state.Error = err.Error()
	}
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_ = json.NewEncoder(conn).Encode(state)
}

func contactVPNPeer(peer, command string, timeout time.Duration) (vpnPeerState, error) {
	var state vpnPeerState
	name, _, err := vpnPipe(peer)
	if err != nil {
		return state, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := winio.DialPipeContext(ctx, name)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) && command == "status" {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if err := json.NewEncoder(conn).Encode(command); err != nil {
		return state, err
	}
	if err := json.NewDecoder(io.LimitReader(conn, 8192)).Decode(&state); err != nil {
		return state, err
	}
	if state.Error != "" {
		return state, errors.New(state.Error)
	}
	return state, nil
}

func lockVPNTransition() (func(), error) {
	root := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(root) {
		return nil, errors.New("LOCALAPPDATA is unavailable")
	}
	dir := filepath.Join(root, "Connect")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "vpn-transition.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	var lock windows.Overlapped
	handle := windows.Handle(file.Fd())
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &lock); err != nil {
		file.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, errors.New("Другое переключение VPN ещё выполняется")
		}
		return nil, err
	}
	return func() {
		_ = windows.UnlockFileEx(handle, 0, 1, 0, &lock)
		_ = file.Close()
	}, nil
}
