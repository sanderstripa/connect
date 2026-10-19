package main

import (
	"connect-installer.local/core/adapters/screencast"
	"connect-installer.local/core/core"
)

func main() {
	a := screencast.New()
	if err := core.Run(a.Config(), a); err != nil {
		panic(err)
	}
}
