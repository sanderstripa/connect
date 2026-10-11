//go:build windows

package main

import (
	"fmt"
	"golang.org/x/sys/windows"
)

func lockChromeHelper() (func(), error) {
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return nil, e
	}
	name, e := windows.UTF16PtrFromString("Local\\ConnectChrome-" + user.User.Sid.String())
	if e != nil {
		return nil, e
	}
	handle, e := windows.CreateMutex(nil, false, name)
	if e != nil {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
		return nil, fmt.Errorf("Расширение уже работает в другом профиле Chrome")
	}
	return func() { windows.CloseHandle(handle) }, nil
}
