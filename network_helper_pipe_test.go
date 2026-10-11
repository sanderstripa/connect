//go:build windows

package main

import (
	"context"
	"fmt"
	"net/rpc"
	"os"
	"testing"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func TestNetworkPipeAllowsNormalDesktopWithToken(t *testing.T) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	pipe := fmt.Sprintf(`\\.\pipe\sudoku-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	listener, err := winio.ListenPipe(pipe, &winio.PipeConfig{SecurityDescriptor: "D:P(A;;GA;;;" + user.User.Sid.String() + ")(A;;GA;;;SY)S:(ML;;NW;;;ME)"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	server := rpc.NewServer()
	if err := server.RegisterName("Network", &NetworkRPC{app: &App{}, token: "test-token"}); err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			server.ServeConn(conn)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := winio.DialPipeContext(ctx, pipe)
	if err != nil {
		t.Fatal(err)
	}
	client := &networkClient{client: rpc.NewClient(conn), token: "wrong"}
	defer client.client.Close()
	if err := client.call("GetState", nil, nil); err == nil {
		t.Fatal("pipe accepted wrong token")
	}
	client.token = "test-token"
	if err := client.call("GetState", nil, nil); err != nil {
		t.Fatal(err)
	}
}
