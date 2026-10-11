//go:build windows

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func needsNetworkHelper() bool { return !windows.GetCurrentProcessToken().IsElevated() }

func startNetworkHelper() (*networkClient, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(secret)
	pipe := `\\.\pipe\connect-network-` + token
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	args, _ := windows.UTF16PtrFromString(fmt.Sprintf("--network-helper %s %s %d", pipe, token, os.Getpid()))
	if err := windows.ShellExecute(0, verb, file, args, nil, windows.SW_HIDE); err != nil {
		return nil, fmt.Errorf("разрешение на запуск сетевой службы: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	for {
		conn, err := winio.DialPipeContext(ctx, pipe)
		if err == nil {
			return &networkClient{client: rpc.NewClient(conn), token: token}, nil
		}
		if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return nil, fmt.Errorf("подключение к сетевой службе: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("подключение к сетевой службе: %w", ctx.Err())
		case <-time.After(30 * time.Millisecond):
		}
	}
}

func runNetworkHelper() bool {
	if len(os.Args) != 5 || os.Args[1] != "--network-helper" {
		return false
	}
	if needsNetworkHelper() {
		return true
	}
	pipe, token := os.Args[2], os.Args[3]
	if len(token) != 64 || pipe != `\\.\pipe\connect-network-`+token {
		return true
	}
	if _, err := hex.DecodeString(token); err != nil {
		return true
	}
	pid, err := strconv.Atoi(os.Args[4])
	if err != nil || pid <= 0 {
		return true
	}
	parent, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return true
	}
	defer windows.CloseHandle(parent)
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return true
	}
	listener, err := winio.ListenPipe(pipe, &winio.PipeConfig{SecurityDescriptor: "D:P(A;;GA;;;" + user.User.Sid.String() + ")(A;;GA;;;SY)S:(ML;;NW;;;ME)", MessageMode: false})
	if err != nil {
		return true
	}
	defer listener.Close()
	service := NewApp(bundledRuntime, "runtime/bin")
	if service.backend == nil {
		return true
	}
	service.backend.Startup(context.Background())
	defer service.ShutdownNow()
	var once sync.Once
	stop := func() { once.Do(func() { _ = listener.Close() }) }
	server := rpc.NewServer()
	if err := server.RegisterName("Network", &NetworkRPC{app: service, token: token, shutdown: stop}); err != nil {
		return true
	}
	go func() { _, _ = windows.WaitForSingleObject(parent, windows.INFINITE); stop() }()
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		go server.ServeConn(conn)
	}
	return true
}
