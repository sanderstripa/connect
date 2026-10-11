package main

import (
	"encoding/json"
	"net"
	"net/rpc"
	"reflect"
	"testing"

	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/core"
)

func TestNetworkHelperAuthenticatesAndRestrictsOperations(t *testing.T) {
	n := &NetworkRPC{app: &App{}, token: "secret"}
	for _, req := range []NetworkRequest{{Method: "GetConfig"}, {Token: "wrong", Method: "GetConfig"}, {Token: "secret", Method: "ServiceStartup"}, {Token: "secret", Method: "GetConfig", Args: []json.RawMessage{json.RawMessage(`1`)}}} {
		if err := n.Invoke(req, &NetworkReply{}); err == nil {
			t.Fatalf("accepted unauthorized or invalid request: %s", req.Method)
		}
	}
	for name := range networkMethods {
		if !reflect.ValueOf(n.app).MethodByName(name).IsValid() {
			t.Fatalf("unknown method %s", name)
		}
	}
}

func TestNetworkHelperDesktopDelegatesOverRPC(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	server := rpc.NewServer()
	if err := server.RegisterName("Network", &NetworkRPC{app: &App{}, token: "secret"}); err != nil {
		t.Fatal(err)
	}
	go server.ServeConn(serverConn)
	client := rpc.NewClient(clientConn)
	defer client.Close()
	app := &App{remote: &networkClient{client: client, token: "secret"}}
	if err := app.SaveConfig(core.AppConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := app.StartProxy(core.StartRequest{WithTun: true}); err != nil {
		t.Fatal(err)
	}
	if app.GetState().LastError != "" {
		t.Fatal("state transport failed")
	}
	if err := app.remote.call("ServiceShutdown", nil, nil); err == nil {
		t.Fatal("exposed lifecycle method")
	}
}
