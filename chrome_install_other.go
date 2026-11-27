//go:build !windows

package main

import "errors"

func (a *App) PrepareChromeExtension() error {
	return errors.New("Chrome extension installation is available on Windows")
}
