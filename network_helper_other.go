//go:build !windows

package main

func runNetworkHelper() bool                      { return false }
func needsNetworkHelper() bool                    { return false }
func startNetworkHelper() (*networkClient, error) { return nil, nil }
