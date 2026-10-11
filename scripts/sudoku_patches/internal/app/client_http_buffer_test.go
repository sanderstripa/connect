package app

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
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
	reader := bufio.NewReader(server)
	req, err := http.ReadRequest(reader)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodConnect {
		t.Fatal(req.Method)
	}
	if reader.Buffered() != len(payload) {
		t.Fatal("test must prefetch the entire TLS payload")
	}
	got := make([]byte, len(payload))
	_, err = io.ReadFull(&httpBufferedConn{Conn: server, reader: reader}, got)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("TLS bytes lost: %x, %v", got, err)
	}
}
