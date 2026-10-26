//go:build !windows

package core

import (
	"context"
	"fmt"
)

func checkTunHTTPS(context.Context, int) error {
	return fmt.Errorf("pinned TUN health check is Windows-only")
}
