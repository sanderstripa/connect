//go:build !windows

package core

import "errors"

func nativeWindowsDefaultRoute(int) (string, int, error) { return "", 0, errors.New("Windows only") }
