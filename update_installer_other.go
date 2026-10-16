//go:build !windows

package main

import "fmt"

func launchUpdateInstaller(string) error { return fmt.Errorf("application updates require Windows") }
