package main

import (
	"connect-installer.local/core/adapters/soulu"
	"connect-installer.local/core/core"
)

func main() {
	a := soulu.New()
	if err := core.Run(a.Config(), a); err != nil {
		panic(err)
	}
}
