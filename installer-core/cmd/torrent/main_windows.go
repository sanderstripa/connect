package main

import (
	"connect-installer.local/core/adapters/torrent"
	"connect-installer.local/core/core"
)

func main() {
	a := torrent.New()
	if err := core.Run(a.Config(), a); err != nil {
		panic(err)
	}
}
