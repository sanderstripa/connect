package main

import (
	"connect-installer.local/core/adapters/screenshot"
	"connect-installer.local/core/core"
)

func main() {
	a := screenshot.New()
	if err := core.Run(a.Config(), a); err != nil {
		panic(err)
	}
}
