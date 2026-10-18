//go:build !windows

package main

func containChromeProcesses() (func(), error) { return func() {}, nil }
