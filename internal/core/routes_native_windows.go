//go:build windows

package core

import (
	"encoding/binary"
	"errors"
	"golang.org/x/sys/windows"
	"net"
	"syscall"
	"unsafe"
)

// MIB_IPFORWARDROW uses fourteen DWORDs. Metric1 already includes the
// interface metric on Windows Vista and newer (Microsoft IP Helper contract).
func selectNativeDefaultRoute(table []byte, excludeIf int, up map[int]bool) (string, int, error) {
	if len(table) < 4 {
		return "", 0, errors.New("truncated IPv4 route table")
	}
	count := int(binary.LittleEndian.Uint32(table[:4]))
	if count > (len(table)-4)/56 {
		return "", 0, errors.New("truncated IPv4 route rows")
	}
	bestMetric := uint32(^uint32(0))
	bestIndex := 0
	gateway := ""
	for i := 0; i < count; i++ {
		row := table[4+i*56 : 4+(i+1)*56]
		index := int(binary.LittleEndian.Uint32(row[16:20]))
		metric := binary.LittleEndian.Uint32(row[36:40])
		if binary.LittleEndian.Uint32(row[0:4]) != 0 || binary.LittleEndian.Uint32(row[4:8]) != 0 || index == excludeIf || !up[index] {
			continue
		}
		hop := net.IP(row[12:16])
		if hop.IsUnspecified() {
			continue
		}
		if bestIndex == 0 || metric < bestMetric {
			bestMetric = metric
			bestIndex = index
			gateway = hop.String()
		}
	}
	if bestIndex == 0 {
		return "", 0, errors.New("physical default route not found")
	}
	return gateway, bestIndex, nil
}

func nativeWindowsDefaultRoute(excludeIf int) (string, int, error) {
	proc := windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetIpForwardTable")
	var size uint32
	code, _, _ := proc.Call(0, uintptr(unsafe.Pointer(&size)), 0)
	if code != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
		return "", 0, syscall.Errno(code)
	}
	for attempt := 0; attempt < 3; attempt++ {
		if size < 4 || size > 64*1024*1024 {
			return "", 0, errors.New("invalid IPv4 route table size")
		}
		table := make([]byte, size)
		code, _, _ = proc.Call(uintptr(unsafe.Pointer(&table[0])), uintptr(unsafe.Pointer(&size)), 0)
		if code == uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			continue
		}
		if code != 0 {
			return "", 0, syscall.Errno(code)
		}
		adapters, err := net.Interfaces()
		if err != nil {
			return "", 0, err
		}
		up := map[int]bool{}
		for _, adapter := range adapters {
			up[adapter.Index] = adapter.Flags&net.FlagUp != 0
		}
		return selectNativeDefaultRoute(table, excludeIf, up)
	}
	return "", 0, errors.New("IPv4 route table kept changing")
}
