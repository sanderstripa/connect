//go:build !windows

package main

func lockChromeHelper() (func(), error) { return func() {}, nil }
