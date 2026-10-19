//go:build windows

package main

import (
	"errors"

	"golang.org/x/sys/windows"
)

type folderSelection struct {
	Path      string `json:"path"`
	Cancelled bool   `json:"cancelled"`
}

func folderDialogResult(path string, err error) (folderSelection, error) {
	if err != nil {
		cancelled := errors.Is(err, windows.ERROR_CANCELLED)
		var hresult interface{ Code() uintptr }
		if errors.As(err, &hresult) && uint32(hresult.Code()) == 0x800704c7 {
			cancelled = true // HRESULT_FROM_WIN32(ERROR_CANCELLED)
		}
		// Wails alpha.74 returns its internal cfd.ErrorCancelled sentinel here.
		// It cannot be imported outside Wails; match its exact message only at
		// this native-dialog boundary, including errors wrapping that sentinel.
		for cause := err; cause != nil; cause = errors.Unwrap(cause) {
			if cause.Error() == "cancelled by user" {
				cancelled = true
				break
			}
		}
		if !cancelled {
			return folderSelection{}, err
		}
		return folderSelection{Cancelled: true}, nil
	}
	return folderSelection{Path: path, Cancelled: path == ""}, nil
}
