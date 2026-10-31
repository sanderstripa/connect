package app

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestHTTPProxyPreservesPrefetchedTunnelBytes(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	payload := []byte{0x16, 0x03, 0x01, 0x00, 0x04, 1, 2, 3, 4}
	go func() {
		_, _ = client.Write(append([]byte("CONNECT auth.openai.com:443 HTTP/1.1\r\nHost: auth.openai.com:443\r\n\r\n"), payload...))
	}()
	_ = server.SetReadDeadline(time.Now().Add(time.Second))
	req, early, err := readProxyRequest(server)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "CONNECT" {
		t.Fatal(req.Method)
	}
	if !bytes.Equal(early, payload) {
		t.Fatalf("TLS bytes lost: %x", early)
	}
}
