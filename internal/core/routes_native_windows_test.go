//go:build windows

package core

import (
	"encoding/binary"
	"testing"
)

func TestNativeRouteSelectionExcludesTunAndUsesCombinedMetric(t *testing.T) {
	table := make([]byte, 4+3*56)
	binary.LittleEndian.PutUint32(table, 3)
	for i, index := range []uint32{7, 9, 11} {
		row := table[4+i*56:]
		copy(row[12:16], []byte{192, 0, 2, 1})
		binary.LittleEndian.PutUint32(row[16:20], index)
		binary.LittleEndian.PutUint32(row[36:40], []uint32{1, 200, 50}[i])
	}
	_, index, err := selectNativeDefaultRoute(table, 7, map[int]bool{7: true, 9: true, 11: true})
	if err != nil || index != 11 {
		t.Fatalf("index=%d err=%v", index, err)
	}
	_, index, err = selectNativeDefaultRoute(table, 7, map[int]bool{7: true, 9: true})
	if err != nil || index != 9 {
		t.Fatalf("down adapter selected: %d %v", index, err)
	}
	if _, _, err = selectNativeDefaultRoute(table[:6], 7, nil); err == nil {
		t.Fatal("truncated table accepted")
	}
}
