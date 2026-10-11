package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNativeFraming(t *testing.T) {
	var buf bytes.Buffer
	want := chromeRequest{ID: 42, Action: "status"}
	if e := writeNativeMessage(&buf, want); e != nil {
		t.Fatal(e)
	}
	var got chromeRequest
	if e := readNativeMessage(&buf, &got); e != nil || got != want {
		t.Fatalf("%+v %v", got, e)
	}
}
func TestNativeRejectsLargeMessage(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(1024*1024+1))
	if readNativeMessage(&buf, &chromeRequest{}) == nil {
		t.Fatal("accepted oversized message")
	}
}
